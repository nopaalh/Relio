package repository

import (
	"regexp"
	"testing"
)

func TestFullUsageQuerySelectsLoaderBackingRecord(t *testing.T) {
	// Boundary fixture: properties written by the loader, not a prepared transport response.
	stored := map[string]string{"available_from": "2026-09-01", "account_id": "C01"}
	predicate := regexp.MustCompile(`p\.([a-z_]+) <= \$as_of`).FindStringSubmatch(fullSourceQuery)
	if len(predicate) != 2 {
		t.Fatal("usage query must bound source availability")
	}
	value, exists := stored[predicate[1]]
	if !exists || value > "2026-09-22" {
		t.Fatal("query excludes an authorized visible loader-created source container", predicate[1])
	}
}
