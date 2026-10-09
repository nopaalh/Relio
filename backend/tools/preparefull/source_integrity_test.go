package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Acceptance checks for existing source guards; original dataset is never modified.
func TestFullSourceChangedChecksumRefusesPreparation(t *testing.T) {
	dir := t.TempDir()
	files, e := os.ReadDir("../../../../Datasets")
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		b, e := os.ReadFile(filepath.Join("../../../../Datasets", f.Name()))
		if e != nil {
			t.Fatal(e)
		}
		if f.Name() == "crm_accounts.csv" {
			b = append(b, '\n')
		}
		if e = os.WriteFile(filepath.Join(dir, f.Name()), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	_, e = readSources(dir, "../../docs/source-audit-snapshot.json")
	if e == nil || !strings.Contains(e.Error(), "checksum") {
		t.Fatalf("changed source not refused by checksum: %v", e)
	}
}
func TestFullSourceBrokenNativeFKRefused(t *testing.T) {
	s, e := readSources("../../../../Datasets", "../../docs/source-audit-snapshot.json")
	if e != nil {
		t.Fatal(e)
	}
	s.rows["crm_deals.csv"][0]["account_id"] = "MISSING_ILLUSTRATION_ID"
	if e = validateSources(s); e == nil || !strings.Contains(e.Error(), "FK") {
		t.Fatalf("broken FK not refused: %v", e)
	}
}
