package repository

import (
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"testing"
)

func TestFullArtifactSealDetectsPayloadReplacement(t *testing.T) {
	a := FullArtifact{Manifest: FullManifest{SchemaVersion: FullSchemaVersion, DatasetVersion: "full:test", Sources: map[string]string{}, SourceCounts: map[string]int{}, Counts: map[string]int{}, DealIDs: []string{}, AccountIDs: []string{}}, Records: []FullRecord{{Kind: "nodes", ID: "account:X", PartitionID: "X|2026-10|nodes", AccountIDs: []string{"X"}, DealIDs: []string{}, AvailableFrom: models.Date("2026-10-01"), Payload: json.RawMessage(`{"node_id":"account:X"}`)}}}
	if err := SealFullArtifact(&a); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFullArtifact(a); err != nil {
		t.Fatal(err)
	}
	a.Records[0].Payload = json.RawMessage(`{"node_id":"account:Y"}`)
	a.Records[0].PayloadHash = fullHash(a.Records[0].Payload)
	if err := ValidateFullArtifact(a); err == nil {
		t.Fatal("replacement bypassed manifest")
	}
}
