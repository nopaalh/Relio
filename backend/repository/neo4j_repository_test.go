package repository

import (
	"context"
	"errors"
	"github.com/nopaalh/Relio/backend/models"
	"strings"
	"testing"
)

var _ DealFactsRepository = (*Neo4jContextRepository)(nil)
var _ GraphRepository = (*Neo4jContextRepository)(nil)
var _ TimelineRepository = (*Neo4jContextRepository)(nil)
var _ EvidenceRepository = (*Neo4jContextRepository)(nil)

func testP04Repo(t *testing.T) (*Neo4jContextRepository, storedContext) {
	raw := readyP04(t)
	return &Neo4jContextRepository{read: func(ctx context.Context, s models.SnapshotContext) (storedContext, error) { return raw, nil }}, raw
}
func TestNeo4jRepositoryGraphBoundsAndFocus(t *testing.T) {
	r, raw := testP04Repo(t)
	s := snapshotP04(t, raw, "2026-09-01")
	ctx := context.Background()
	g, err := r.ReadGraph(ctx, "DL-004", s, models.GraphOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Nodes) != 4 {
		t.Fatalf("default account-root nodes %d", len(g.Nodes))
	}
	deep, err := r.ReadGraph(ctx, "DL-004", s, models.GraphOptions{Depth: 2})
	if err != nil || len(deep.Nodes) != 6 {
		t.Fatalf("depth2: %v nodes %d", err, len(deep.Nodes))
	}
	tiny, err := r.ReadGraph(ctx, "DL-004", s, models.GraphOptions{Depth: 2, MaxNodes: 3, MaxEdges: 1})
	if err != nil || !tiny.Bounds.Truncated || len(tiny.Nodes) > 3 || len(tiny.Edges) > 1 {
		t.Fatal("bounds not enforced")
	}
	_, err = r.ReadGraph(ctx, "DL-004", s, models.GraphOptions{Depth: 3})
	requireCode(t, err, InvalidQuery)
	focus := "event:interaction:I0335"
	_, err = r.ReadGraph(ctx, "DL-004", s, models.GraphOptions{FocusEventID: &focus})
	requireCode(t, err, NotVisible)
	focus = "unknown"
	_, err = r.ReadGraph(ctx, "DL-004", s, models.GraphOptions{FocusEventID: &focus})
	requireCode(t, err, NotFound)
}

func TestNeo4jFocusedGraphRetainsEventUnderCap(t *testing.T) {
	r, raw := testP04Repo(t)
	s := snapshotP04(t, raw, "2026-09-01")
	focus := "event:interaction:I0314"
	g, err := r.ReadGraph(context.Background(), "DL-004", s, models.GraphOptions{FocusEventID: &focus, MaxNodes: 2})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range g.Nodes {
		if n.NodeID == focus {
			found = true
		}
	}
	if !found {
		t.Fatal("successful focused graph omitted focus")
	}
}
func TestNeo4jRepositoryTimelineCursor(t *testing.T) {
	r, raw := testP04Repo(t)
	s := snapshotP04(t, raw, "2026-09-22")
	ctx := context.Background()
	a, err := r.ReadTimeline(ctx, "DL-004", s, models.TimelineOptions{Limit: 1})
	if err != nil || len(a.Events) != 1 || a.Events[0].EventID != "event:interaction:I0284" || a.Bounds.NextCursor == nil {
		t.Fatal("first page")
	}
	b, err := r.ReadTimeline(ctx, "DL-004", s, models.TimelineOptions{Limit: 1, Cursor: a.Bounds.NextCursor})
	if err != nil || b.Events[0].EventID != "event:interaction:I0314" {
		t.Fatal("next page")
	}
	_, err = r.ReadTimeline(ctx, "DL-004", snapshotP04(t, raw, "2026-09-01"), models.TimelineOptions{Limit: 1, Cursor: a.Bounds.NextCursor})
	requireCode(t, err, ContextMismatch)
	_, err = r.ReadTimeline(ctx, "DL-004", s, models.TimelineOptions{Limit: 1, Cursor: a.Bounds.NextCursor, EventTypes: []string{"email"}})
	requireCode(t, err, InvalidQuery)
	f, err := r.ReadTimeline(ctx, "DL-004", s, models.TimelineOptions{EventTypes: []string{"email"}})
	if err != nil || len(f.Events) != 1 || f.Events[0].EventID != "event:interaction:I0335" {
		t.Fatal("type filter")
	}
	invalid := "bad"
	_, err = r.ReadTimeline(ctx, "DL-004", s, models.TimelineOptions{Cursor: &invalid})
	requireCode(t, err, InvalidQuery)
}
func TestNeo4jRepositoryDealAndEvidenceErrors(t *testing.T) {
	r, raw := testP04Repo(t)
	ctx := context.Background()
	early := snapshotP04(t, raw, "2026-08-01")
	_, err := r.FindFacts(ctx, "DL-004", early)
	requireCode(t, err, NotVisible)
	s := snapshotP04(t, raw, "2026-09-01")
	_, err = r.FindFacts(ctx, "DL-005", s)
	requireCode(t, err, DataNotReady)
	_, err = r.ReadEvidence(ctx, "ev:interactions.jsonl:I0335:isi", s)
	requireCode(t, err, NotVisible)
	_, err = r.ReadEvidence(ctx, "missing", s)
	requireCode(t, err, NotFound)
	empty, _ := NewSnapshotContext("2026-09-01", raw.Manifest.DatasetVersion, models.AccessScope{})
	_, err = r.FindFacts(ctx, "DL-004", empty)
	requireCode(t, err, AccessDenied)
	d, err := r.FindFacts(ctx, "DL-004", snapshotP04(t, raw, "2026-10-01"))
	if err != nil || *d.Deal.PotentialACVIDR.Value != 147000000 {
		t.Fatal("deal facts")
	}
}
func TestNeo4jRepositoryCancellationAndDriverFailure(t *testing.T) {
	r, raw := testP04Repo(t)
	s := snapshotP04(t, raw, "2026-09-01")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := r.FindFacts(ctx, "DL-004", s)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("lost cancellation")
	}
	r.read = func(context.Context, models.SnapshotContext) (storedContext, error) {
		return storedContext{}, errors.New("secret-password")
	}
	_, err = r.FindFacts(context.Background(), "DL-004", s)
	requireCode(t, err, QueryFailed)
	if strings.Contains(err.Error(), "secret") {
		t.Fatal("cause exposed")
	}
	r.read = nil
	_, err = r.FindFacts(context.Background(), "DL-004", s)
	requireCode(t, err, DataNotReady)
}
func TestNeo4jQueriesReadOnlyAndParameterized(t *testing.T) {
	for _, q := range []string{neo4jManifestQuery, neo4jRecordsQuery} {
		u := strings.ToUpper(q)
		for _, verb := range []string{"CREATE ", "MERGE ", "DELETE ", " SET "} {
			if strings.Contains(u, verb) {
				t.Fatal("runtime write")
			}
		}
		if !strings.Contains(q, "$version") {
			t.Fatal("unbound namespace")
		}
	}
}
func TestNeo4jConfigValidation(t *testing.T) {
	for _, c := range []Neo4jConfig{{}, {URI: "bolt://localhost", Username: "u", Password: "p", Database: "neo4j"}, {URI: "neo4j+s://example.invalid", Username: "u", Database: "neo4j"}} {
		_, err := NewNeo4jContextRepository(context.Background(), c)
		requireCode(t, err, DataNotReady)
	}
}
