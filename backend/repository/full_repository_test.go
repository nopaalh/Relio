package repository

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/nopaalh/Relio/backend/models"
	"testing"
)

// Contract illustration, not native dataset findings. Only network I/O is doubled.
func fullFixture(t *testing.T) (FullArtifact, *Neo4jFullContextRepository) {
	t.Helper()
	a := FullArtifact{Manifest: FullManifest{SchemaVersion: FullSchemaVersion, DatasetVersion: "full:fixture", Sources: map[string]string{}, SourceCounts: map[string]int{}, DealIDs: []string{"D1"}, AccountIDs: []string{"A1", "A2"}, DealAccountIDs: map[string]string{"D1": "A1"}, DealCreatedAt: map[string]models.Date{"D1": "2026-09-01"}}}
	add := func(kind, id, account string, at models.Date, v any) {
		b, _ := json.Marshal(v)
		a.Records = append(a.Records, FullRecord{Kind: kind, ID: id, PartitionID: account + "|" + kind, AccountIDs: []string{account}, DealIDs: []string{}, AvailableFrom: at, Payload: b})
	}
	ev := models.Evidence{EvidenceID: "proof", SourceFile: "fixture.csv", SourceRecordID: "row1", RecordIDKind: "native", SourceKey: map[string]string{"id": "row1"}, SourceChecksum: "fixture", SourceField: "created", SourceDate: func() *models.Date { d := models.Date("2026-09-01"); return &d }(), TemporalBasis: "event", ContentExcerpt: "illustration", ScopeKind: "account", AccountIDs: []string{"A1"}, DealIDs: []string{}, EventIDs: []string{"event"}, EdgeIDs: []string{"edge"}, OccurrenceIDs: []string{}}
	add("evidence", "proof", "A1", "2026-09-01", ev)
	f := models.Fact[string]{Value: func() *string { x := "Current"; return &x }(), State: "known", TemporalBasis: "snapshot", EvidenceIDs: []string{"proof"}, Limitations: []string{}}
	d := models.DealFacts{DealID: "D1", AccountID: "A1", Stage: f, RecordEvidenceIDs: []string{"proof"}}
	add("deals", "D1", "A1", "2026-09-01", d)
	for _, id := range []string{"account:A1", "deal:D1", "event"} {
		add("nodes", id, "A1", "2026-09-01", models.Node{NodeID: id, EntityID: id, NodeType: "illustration", Label: f, Validity: models.Validity{TemporalBasis: "event"}, EvidenceIDs: []string{"proof"}, EventIDs: []string{"event"}})
	}
	add("edges", "edge", "A1", "2026-09-01", models.Edge{EdgeID: "edge", Source: "account:A1", Target: "event", Validity: models.Validity{TemporalBasis: "event"}, EvidenceIDs: []string{"proof"}, EventIDs: []string{"event"}})
	add("events", "event", "A1", "2026-09-01", models.Event{EventID: "event", EventAt: "2026-09-01", ScopeKind: "account", AccountIDs: []string{"A1"}, DealIDs: []string{}, NodeIDs: []string{"event"}, EdgeIDs: []string{"edge"}, EvidenceIDs: []string{"proof"}, Actors: []models.ParticipantRef{}, Targets: []models.ParticipantRef{}})
	add("evidence", "foreign", "A2", "2026-09-01", models.Evidence{EvidenceID: "foreign", ScopeKind: "account", AccountIDs: []string{"A2"}})
	ev.EvidenceID = "future"
	ev.SourceDate = func() *models.Date { d := models.Date("2026-10-01"); return &d }()
	add("evidence", "future", "A1", "2026-10-01", ev)
	if err := SealFullArtifact(&a); err != nil {
		t.Fatal(err)
	}
	r := &Neo4jFullContextRepository{read: func(_ context.Context, q fullQuery) (fullSelection, error) {
		return fullSelection{Manifest: a.Manifest, Partitions: a.Partitions, Records: a.Records}, nil
	}}
	return a, r
}
func fullSnapshot(t *testing.T, at models.Date) models.SnapshotContext {
	t.Helper()
	s, e := NewSnapshotContext(at, "full:fixture", models.AccessScope{AllowedAccountIDs: []string{"A1"}, AllowedDealIDs: []string{"D1"}})
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func requireFullCode(t *testing.T, err error, code ErrorCode) {
	t.Helper()
	var e *RepositoryError
	if !errors.As(err, &e) || e.Code != code {
		t.Fatalf("want %s got %v", code, err)
	}
}
func TestFullRepositorySnapshotScope(t *testing.T) {
	_, r := fullFixture(t)
	s := fullSnapshot(t, "2026-09-22")
	d, e := r.FindFacts(context.Background(), "D1", s)
	if e != nil {
		t.Fatal(e)
	}
	if d.Deal.Stage.Value != nil || d.Deal.Stage.State != "snapshot_only" {
		t.Fatal("current stage leaked into history")
	}
	g, e := r.ReadGraph(context.Background(), "D1", s, models.GraphOptions{FocusEventID: func() *string { x := "event"; return &x }(), MaxNodes: 1})
	if e != nil {
		t.Fatal(e)
	}
	if len(g.Nodes) != 1 || g.Nodes[0].NodeID != "event" || !g.Bounds.Truncated {
		t.Fatal("focus lost under cap")
	}
	tl, e := r.ReadTimeline(context.Background(), "D1", s, models.TimelineOptions{})
	if e != nil || len(tl.Events) != 1 {
		t.Fatalf("timeline: %v %v", tl, e)
	}
	ev, e := r.ReadEvidence(context.Background(), "proof", s)
	if e != nil || ev.Meta.ContextID != d.Meta.ContextID || g.Meta.ContextID != tl.Meta.ContextID {
		t.Fatal("inconsistent context")
	}
	_, e = r.ReadEvidence(context.Background(), "future", s)
	requireFullCode(t, e, NotVisible)
	_, e = r.ReadEvidence(context.Background(), "foreign", s)
	requireFullCode(t, e, AccessDenied)
	_, e = r.FindFacts(context.Background(), "unknown", s)
	requireFullCode(t, e, NotFound)
	_, e = r.FindFacts(context.Background(), "D1", fullSnapshot(t, "2026-08-01"))
	requireFullCode(t, e, NotVisible)
}
func TestFullRepositoryRejectsMissingNativeDeal(t *testing.T) {
	a, r := fullFixture(t)
	records := []FullRecord{}
	for _, x := range a.Records {
		if x.Kind != "deals" {
			records = append(records, x)
		}
	}
	r.read = func(context.Context, fullQuery) (fullSelection, error) {
		return fullSelection{Manifest: a.Manifest, Partitions: a.Partitions, Records: records}, nil
	}
	_, e := r.FindFacts(context.Background(), "D1", fullSnapshot(t, MaxAsOf))
	requireFullCode(t, e, DataNotReady)
}
