// One-off preparation tool. Never called by server startup or HTTP handlers.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type config struct{ uri, username, password, database, version string }

func parseConfig(text string) (config, error) {
	values := map[string]string{}
	keys := []string{"NEO4J_URI", "NEO4J_USERNAME", "NEO4J_PASSWORD", "NEO4J_DATABASE", "RELIO_DATASET_VERSION"}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return config{}, errors.New("malformed .env line")
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		known := false
		for _, allowed := range keys {
			if key == allowed {
				known = true
			}
		}
		if !known {
			continue
		}
		if _, duplicate := values[key]; duplicate {
			return config{}, fmt.Errorf("duplicate configuration key: %s", key)
		}
		if strings.HasPrefix(value, "\"") || strings.HasPrefix(value, "'") {
			if len(value) < 2 || value[len(value)-1] != value[0] {
				return config{}, fmt.Errorf("unclosed quote: %s", key)
			}
			value = value[1 : len(value)-1]
		}
		values[key] = value
	}
	for _, key := range keys {
		if values[key] == "" || values[key] == "PASSWORD_AURA_KAMU" || strings.Contains(values[key], "xxxxxxxx") {
			return config{}, fmt.Errorf("missing/placeholder configuration: %s", key)
		}
	}
	u, err := url.Parse(values["NEO4J_URI"])
	if err != nil || u.Scheme != "neo4j+s" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return config{}, errors.New("invalid NEO4J_URI; expected neo4j+s URI without embedded credentials")
	}
	return config{values[keys[0]], values[keys[1]], values[keys[2]], values[keys[3]], values[keys[4]]}, nil
}

// shortcut: prepared artifacts use one complete statement per line; reject
// multiline Cypher rather than guessing how arbitrary scripts should split.
func statements(text string) ([]string, error) {
	queries := []string{}
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		var quote rune
		escaped, terminated := false, false
		for j, r := range line {
			if escaped {
				escaped = false
				continue
			}
			if quote != 0 {
				if r == '\\' {
					escaped = true
				} else if r == quote {
					quote = 0
				}
				continue
			}
			if r == '"' || r == '\'' || r == '`' {
				quote = r
			}
			if r == ';' {
				if j != len(line)-1 {
					return nil, fmt.Errorf("multiple statements on line %d", i+1)
				}
				terminated = true
			}
		}
		if quote != 0 || !terminated {
			return nil, fmt.Errorf("incomplete statement on line %d", i+1)
		}
		queries = append(queries, strings.TrimSuffix(line, ";"))
	}
	if len(queries) == 0 {
		return nil, errors.New("empty Cypher file")
	}
	return queries, nil
}

func safeError(stage string, err error) error {
	var dbErr *neo4j.Neo4jError
	if errors.As(err, &dbErr) {
		return fmt.Errorf("%s failed (%s); raw error omitted", stage, dbErr.Code)
	}
	return fmt.Errorf("%s failed; raw error omitted to protect credentials/payload", stage)
}

func run() error {
	apply := flag.Bool("apply", false, "write reviewed schema/seed to configured database")
	inspect := flag.Bool("inspect", false, "read-only connection/database preflight")
	flag.Parse()
	data, err := os.ReadFile(".env")
	if err != nil {
		return errors.New("cannot read backend/.env; run from backend directory")
	}
	c, err := parseConfig(string(data))
	if err != nil {
		return err
	}
	manifestBytes, err := os.ReadFile("database/neo4j/manifest-p04.json")
	if err != nil {
		return errors.New("cannot read preparation manifest")
	}
	var manifest struct {
		Version string `json:"dataset_version"`
		Hash    string `json:"manifest_hash"`
	}
	if json.Unmarshal(manifestBytes, &manifest) != nil || manifest.Version != c.version || manifest.Hash == "" {
		return errors.New("configured dataset version does not match preparation manifest")
	}
	files := [][]string{}
	for _, name := range []string{"schema.cypher", "seed-p04.cypher"} {
		bytes, err := os.ReadFile(filepath.Join("database/neo4j", name))
		if err != nil {
			return errors.New("cannot read " + name)
		}
		qs, err := statements(string(bytes))
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		files = append(files, qs)
	}
	if len(files[0]) != 2 || len(files[1]) != 49 {
		return errors.New("unexpected prepared statement count; review artifacts first")
	}
	u, _ := url.Parse(c.uri)
	fmt.Printf("Target host=%s database=%s; schema=%d seed=%d statements\n", u.Host, c.database, len(files[0]), len(files[1]))
	if !*apply && !*inspect {
		fmt.Println("Dry run only. Use -inspect for read-only preflight or -apply for seed loading.")
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	driver, err := neo4j.NewDriverWithContext(c.uri, neo4j.BasicAuth(c.username, c.password, ""))
	if err != nil {
		return safeError("driver", err)
	}
	defer driver.Close(context.Background())
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return safeError("connectivity", err)
	}
	admin := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "system", AccessMode: neo4j.AccessModeRead})
	result, err := admin.Run(ctx, "SHOW DATABASES YIELD name RETURN name", nil)
	if err != nil {
		_ = admin.Close(ctx)
		return safeError("database inspection", err)
	}
	rows, err := result.Collect(ctx)
	_ = admin.Close(ctx)
	if err != nil {
		return safeError("database inspection", err)
	}
	found := false
	for _, row := range rows {
		name, _ := row.Get("name")
		fmt.Printf("Visible database: %v\n", name)
		if name == c.database {
			found = true
		}
	}
	if !found {
		return errors.New("NEO4J_DATABASE is not a visible database; no writes performed")
	}
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database, AccessMode: neo4j.AccessModeWrite})
	defer session.Close(context.Background())
	result, err = session.Run(ctx, "MATCH (d:RelioDataset {dataset_version:$version}) RETURN d.manifest_hash AS hash", map[string]any{"version": c.version})
	if err != nil {
		return safeError("manifest inspection", err)
	}
	rows, err = result.Collect(ctx)
	if err != nil {
		return safeError("manifest inspection", err)
	}
	for _, row := range rows {
		hash, _ := row.Get("hash")
		if hash != manifest.Hash {
			return errors.New("existing manifest hash differs; no writes performed")
		}
	}
	if !*apply {
		fmt.Println("Read-only preflight PASS; no writes performed.")
		return nil
	}
	for i, query := range files[0] {
		result, err := session.Run(ctx, query, nil)
		if err != nil {
			return safeError(fmt.Sprintf("schema statement %d", i+1), err)
		}
		if _, err := result.Consume(ctx); err != nil {
			return safeError("schema consume", err)
		}
	}
	tx, err := session.BeginTransaction(ctx)
	if err != nil {
		return safeError("seed transaction", err)
	}
	defer tx.Close(context.Background())
	for i, query := range files[1] {
		result, err := tx.Run(ctx, query, nil)
		if err != nil {
			return safeError(fmt.Sprintf("seed statement %d", i+1), err)
		}
		rows, err := result.Collect(ctx)
		if err != nil {
			return safeError(fmt.Sprintf("seed statement %d", i+1), err)
		}
		if i == len(files[1])-1 {
			if len(rows) != 1 {
				return errors.New("seed readiness returned no unique row; seed transaction rolled back")
			}
			m := rows[0].AsMap()
			if m["ready"] != true || m["actual"] != int64(46) || m["links"] != int64(8) || m["dataset_version"] != c.version {
				return errors.New("seed readiness invalid; seed transaction rolled back")
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return safeError("seed commit", err)
	}
	fmt.Println("Seed committed: ready=true actual=46 links=8. Run TestNeo4jP04Live next.")
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
