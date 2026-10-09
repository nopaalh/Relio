package repository

import (
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"testing"
)

func TestFullPartitionBindingIncludesAvailabilityAndScope(t *testing.T) {
	a := FullArtifact{Manifest: FullManifest{SchemaVersion: FullSchemaVersion, DatasetVersion: "full:test", Sources: map[string]string{}, SourceCounts: map[string]int{}, DealIDs: []string{}, AccountIDs: []string{"A"}}, Records: []FullRecord{{Kind: "nodes", ID: "account:A", PartitionID: "A|2026-10|nodes", AccountIDs: []string{"A"}, DealIDs: []string{}, AvailableFrom: models.Date("2026-10-01"), Payload: json.RawMessage(`{"node_id":"account:A"}`)}}}
	if err := SealFullArtifact(&a); err != nil {
		t.Fatal(err)
	}
	if err := verifyFullSelection(a.Manifest, a.Partitions, a.Records); err != nil {
		t.Fatal(err)
	}
	a.Records[0].AvailableFrom = "2024-01-01"
	a.Records[0].AccountIDs = []string{"B"}
	if err := verifyFullSelection(a.Manifest, a.Partitions, a.Records); err == nil {
		t.Fatal("metadata change bypassed partition binding")
	}
}
func TestFullQuerySelectsScopeAndTime(t *testing.T) {
	s, _ := NewSnapshotContext("2026-09-01", "full:test", models.AccessScope{AllowedAccountIDs: []string{"A"}, AllowedDealIDs: []string{"D"}, AllowedAnalogAccountIDs: []string{"B"}})
	q := fullQueryFor(s, "graph")
	if len(q.Accounts) != 1 || q.Accounts[0] != "A" || q.Company || q.AsOf != "2026-09-01" {
		t.Fatal("graph broadened access")
	}
	q = fullQueryFor(s, "actions")
	if len(q.Accounts) != 2 {
		t.Fatal("explicit analog permission omitted")
	}
}
