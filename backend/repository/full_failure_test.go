package repository

import (
	"context"
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"testing"
)

func TestFullUsageKnownMissingContainerIsDataNotReady(t *testing.T) {
	a, r := fullFixture(t)
	a.SourcePartitions = []FullSourcePartition{{ID: "usage-source:A1|2026-09", SourceFile: "product_usage_daily.csv", AccountID: "A1", Date: "2026-09-01", Rows: []map[string]string{}, Keys: []map[string]string{}}}
	if e := SealFullArtifact(&a); e != nil {
		t.Fatal(e)
	}
	r.read = func(context.Context, fullQuery) (fullSelection, error) {
		return fullSelection{Manifest: a.Manifest, Partitions: a.Partitions, Records: a.Records}, nil
	}
	_, e := r.ReadEvidence(context.Background(), "ev:usage:A1|2026-09", fullSnapshot(t, "2026-09-22"))
	requireFullCode(t, e, DataNotReady)
}
func TestFullActionsPropagateMalformedAuthorizedAnalog(t *testing.T) {
	a, r := fullFixture(t)
	a.Records = append(a.Records, FullRecord{Kind: "nodes", ID: "bad", PartitionID: "A2|nodes", AccountIDs: []string{"A2"}, DealIDs: []string{}, AvailableFrom: "2026-09-01", Payload: json.RawMessage(`{"node_id":"bad","label":{"value":{"bad":true}}}`)})
	if e := SealFullArtifact(&a); e != nil {
		t.Fatal(e)
	}
	r.read = func(context.Context, fullQuery) (fullSelection, error) {
		return fullSelection{Manifest: a.Manifest, Partitions: a.Partitions, Records: a.Records}, nil
	}
	s, _ := NewSnapshotContext(MaxAsOf, a.Manifest.DatasetVersion, models.AccessScope{AllowedAccountIDs: []string{"A1"}, AllowedDealIDs: []string{"D1"}, AllowedAnalogAccountIDs: []string{"A2"}})
	_, e := r.ReadActionCandidates(context.Background(), "D1", s, models.CandidateOptions{RelevanceRuleVersion: FullRelevanceVersion})
	requireFullCode(t, e, DataNotReady)
}
