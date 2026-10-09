package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/nopaalh/Relio/backend/repository"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type sources struct {
	rows   map[string][]map[string]string
	hashes map[string]string
	counts map[string]int
}

func parseCSV(r io.Reader) ([]map[string]string, error) {
	c := csv.NewReader(r)
	head, err := c.Read()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for i, h := range head {
		h = strings.TrimPrefix(h, "\ufeff")
		if h == "" || seen[h] {
			return nil, fmt.Errorf("invalid CSV header")
		}
		head[i] = h
		seen[h] = true
	}
	rows := []map[string]string{}
	for {
		values, err := c.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		row := map[string]string{}
		for i, h := range head {
			row[h] = values[i]
		}
		rows = append(rows, row)
	}
	return rows, nil
}
func parseJSONL(r io.Reader) ([]map[string]string, error) {
	scan := bufio.NewScanner(r)
	scan.Buffer(make([]byte, 4096), 4*1024*1024)
	rows := []map[string]string{}
	for scan.Scan() {
		var row map[string]string
		if err := json.Unmarshal(scan.Bytes(), &row); err != nil {
			return nil, err
		}
		if row == nil {
			return nil, fmt.Errorf("null JSONL record")
		}
		rows = append(rows, row)
	}
	return rows, scan.Err()
}
func uniqueRows(rows []map[string]string, keys []string) error {
	seen := map[string]bool{}
	for i, row := range rows {
		tuple := []string{}
		for _, key := range keys {
			v, ok := row[key]
			if !ok {
				return fmt.Errorf("missing column %s", key)
			}
			tuple = append(tuple, v)
		}
		id := repository.FullDigestJSON(tuple)
		if seen[id] {
			return fmt.Errorf("duplicate key at record %d", i+1)
		}
		seen[id] = true
	}
	return nil
}
func readSources(dir, auditPath string) (sources, error) {
	s := sources{map[string][]map[string]string{}, map[string]string{}, map[string]int{}}
	auditData, err := os.ReadFile(auditPath)
	if err != nil {
		return s, err
	}
	var audit struct {
		Files map[string]struct {
			Hash    string   `json:"sha256"`
			Rows    int      `json:"rows"`
			Keys    []string `json:"key_columns_checked"`
			Columns []string `json:"columns"`
		}
	}
	if err = json.Unmarshal(auditData, &audit); err != nil {
		return s, err
	}
	if len(audit.Files) != 15 {
		return s, fmt.Errorf("expected 15 audited files")
	}
	for name, expected := range audit.Files {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return s, fmt.Errorf("%s: cannot read", name)
		}
		// Audit hashes are hashes of file bytes, not JSON encoding.
		actual := hashBytes(b)
		if actual != expected.Hash {
			return s, fmt.Errorf("%s: source checksum mismatch", name)
		}
		var rows []map[string]string
		if strings.HasSuffix(name, ".csv") {
			rows, err = parseCSV(strings.NewReader(string(b)))
		} else {
			rows, err = parseJSONL(strings.NewReader(string(b)))
		}
		if err != nil {
			return s, fmt.Errorf("%s: parse failure: %w", name, err)
		}
		if len(rows) != expected.Rows {
			return s, fmt.Errorf("%s: source count mismatch", name)
		}
		if len(expected.Keys) > 0 {
			if err := uniqueRows(rows, expected.Keys); err != nil {
				return s, fmt.Errorf("%s: %w", name, err)
			}
		}
		for _, row := range rows {
			for _, column := range expected.Columns {
				if _, ok := row[column]; !ok {
					return s, fmt.Errorf("%s: required header absent: %s", name, column)
				}
			}
		}
		s.rows[name] = rows
		s.hashes[name] = actual
		s.counts[name] = len(rows)
	}
	return s, validateSources(s)
}
