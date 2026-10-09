package repository

import (
	"context"
	"github.com/nopaalh/Relio/backend/models"
	"os"
	"testing"
	"time"
)

// Explicit opt-in and read-only: this test never loads seed or changes schema.
func TestNeo4jP04Live(t *testing.T) {
	if os.Getenv("RELIO_NEO4J_INTEGRATION") != "1" {
		t.Skip("Aura live not enabled; seed/config must be prepared explicitly")
	}
	for _, key := range []string{"NEO4J_URI", "NEO4J_USERNAME", "NEO4J_PASSWORD", "NEO4J_DATABASE", "RELIO_DATASET_VERSION"} {
		if os.Getenv(key) == "" {
			t.Fatalf("required environment key absent: %s", key)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	r, err := NewNeo4jContextRepository(ctx, Neo4jConfig{URI: os.Getenv("NEO4J_URI"), Username: os.Getenv("NEO4J_USERNAME"), Password: os.Getenv("NEO4J_PASSWORD"), Database: os.Getenv("NEO4J_DATABASE")})
	if err != nil {
		t.Fatalf("Aura connect: %v", err)
	}
	defer r.Close(context.Background())
	access := models.AccessScope{AllowedAccountIDs: []string{"P04"}, AllowedDealIDs: []string{"DL-004"}}
	for _, tt := range []struct {
		date  string
		count int
	}{{"2026-09-01", 2}, {"2026-09-22", 3}, {"2026-10-01", 3}} {
		t.Run(tt.date, func(t *testing.T) {
			s, err := NewSnapshotContext(models.Date(tt.date), os.Getenv("RELIO_DATASET_VERSION"), access)
			if err != nil {
				t.Fatal(err)
			}
			d, err := r.FindFacts(ctx, "DL-004", s)
			if err != nil {
				t.Fatal(err)
			}
			if tt.date != "2026-10-01" && d.Deal.Stage.Value != nil {
				t.Fatal("historical stage leaked")
			}
			timeline, err := r.ReadTimeline(ctx, "DL-004", s, models.TimelineOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if len(timeline.Events) != tt.count {
				t.Fatal("event count differs")
			}
			graph, err := r.ReadGraph(ctx, "DL-004", s, models.GraphOptions{Depth: 2})
			if err != nil {
				t.Fatal(err)
			}
			if graph.Meta.ContextID != timeline.Meta.ContextID || d.Meta.ContextID != timeline.Meta.ContextID {
				t.Fatal("context mismatch")
			}
			for _, event := range timeline.Events {
				for _, id := range event.EvidenceIDs {
					e, err := r.ReadEvidence(ctx, id, s)
					if err != nil {
						t.Fatal(err)
					}
					if e.Meta.ContextID != s.ContextID {
						t.Fatal("evidence context mismatch")
					}
				}
			}
			_, err = r.ReadEvidence(ctx, "ev:interactions.jsonl:I0335:isi", s)
			if tt.date == "2026-09-01" {
				requireCode(t, err, NotVisible)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
	s, _ := NewSnapshotContext("2026-09-01", os.Getenv("RELIO_DATASET_VERSION"), models.AccessScope{})
	_, err = r.FindFacts(ctx, "DL-004", s)
	requireCode(t, err, AccessDenied)
}
