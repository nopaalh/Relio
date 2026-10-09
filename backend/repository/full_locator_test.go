package repository

import (
	"context"
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"testing"
)

func TestFullHistoricalLocatorDoesNotRevealFutureEnd(t *testing.T) {
	a, r := fullFixture(t)
	for i := range a.Records {
		if a.Records[i].ID != "proof" {
			continue
		}
		var ev models.Evidence
		_ = json.Unmarshal(a.Records[i].Payload, &ev)
		ev.SourceFile = "contact_employment_history.csv"
		ev.SourceKey = map[string]string{"contact_id": "illustration", "mulai": "2026-09-01", "selesai": "2026-10-01"}
		a.Records[i].Payload, _ = json.Marshal(ev)
	}
	if e := SealFullArtifact(&a); e != nil {
		t.Fatal(e)
	}
	r.read = func(context.Context, fullQuery) (fullSelection, error) {
		return fullSelection{Manifest: a.Manifest, Partitions: a.Partitions, Records: a.Records}, nil
	}
	got, e := r.ReadEvidence(context.Background(), "proof", fullSnapshot(t, "2026-09-22"))
	if e != nil {
		t.Fatal(e)
	}
	if _, exists := got.Evidence.SourceKey["selesai"]; exists {
		t.Fatal("future interval end leaked through composite source locator")
	}
	current, e := r.ReadEvidence(context.Background(), "proof", fullSnapshot(t, models.Date("2026-10-01")))
	if e != nil || current.Evidence.SourceKey["selesai"] != "2026-10-01" {
		t.Fatal("projection mutated immutable source")
	}
}
