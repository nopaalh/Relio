package repository

import (
	"context"
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"testing"
)

func TestFullActionsNoPaddingAndAtomicSelection(t *testing.T) {
	a, r := fullFixture(t)
	s := fullSnapshot(t, MaxAsOf)
	empty, e := r.ReadActionCandidates(context.Background(), "D1", s, models.CandidateOptions{RelevanceRuleVersion: "relio-relevance-v1"})
	if e != nil || len(empty.Items) != 0 || empty.TotalValidCount == nil || *empty.TotalValidCount != 0 {
		t.Fatalf("padded candidates: %v %v", empty, e)
	}
	// Illustration price request and an approved precedent, not dataset findings.
	for i := range a.Records {
		if a.Records[i].ID == "proof" {
			var e models.Evidence
			_ = json.Unmarshal(a.Records[i].Payload, &e)
			e.SourceFile = "interactions.jsonl"
			e.SourceField = "isi"
			e.ContentExcerpt = "harga terlalu mahal"
			a.Records[i].Payload, _ = json.Marshal(e)
		}
	}
	add := func(kind, id string, v any) {
		p, _ := json.Marshal(v)
		a.Records = append(a.Records, FullRecord{Kind: kind, ID: id, PartitionID: "A1|" + kind, AccountIDs: []string{"A1"}, DealIDs: []string{}, AvailableFrom: "2026-09-01", Payload: p})
	}
	tpl := models.ActionTemplate{ActionID: "action:discount:1500bps", ActionType: "discount", Definition: "illustration", TaxonomyVersion: "relio-actions-v1"}
	add("templates", tpl.ActionID, tpl)
	o := models.ActionOccurrence{OccurrenceID: "occ", ActionID: tpl.ActionID, EventID: "event", EventAt: "2026-09-01", AccountIDs: []string{"A1"}, DealIDs: []string{}, EvidenceIDs: []string{"proof"}, NodeIDs: []string{"event"}, EdgeIDs: []string{}, Targets: []models.ParticipantRef{}, Status: models.Fact[string]{Value: func() *string { x := "approved"; return &x }(), State: "known", TemporalBasis: "event", EvidenceIDs: []string{"proof"}, Limitations: []string{}}}
	add("occurrences", o.OccurrenceID, o)
	if e := SealFullArtifact(&a); e != nil {
		t.Fatal(e)
	}
	r.read = func(context.Context, fullQuery) (fullSelection, error) {
		return fullSelection{Manifest: a.Manifest, Partitions: a.Partitions, Records: a.Records}, nil
	}
	got, e := r.ReadActionCandidates(context.Background(), "D1", s, models.CandidateOptions{RelevanceRuleVersion: "relio-relevance-v1"})
	if e != nil || len(got.Items) != 1 || got.ReturnedCount != 1 {
		t.Fatalf("one actual candidate: %v %v", got, e)
	}
	selected, e := r.ReadSelectedActions(context.Background(), "D1", s, []string{tpl.ActionID}, "relio-relevance-v1")
	if e != nil || len(selected.Evidence) != 1 {
		t.Fatalf("complete proof: %v %v", selected, e)
	}
	_, e = r.ReadSelectedActions(context.Background(), "D1", s, []string{tpl.ActionID, "invalid"}, "relio-relevance-v1")
	requireFullCode(t, e, InvalidSelection)
	_, e = r.ReadSelectedActions(context.Background(), "D1", s, []string{tpl.ActionID, tpl.ActionID}, "relio-relevance-v1")
	requireFullCode(t, e, InvalidSelection)
	r.MaxSelectedEvidenceBytes = 1
	_, e = r.ReadSelectedActions(context.Background(), "D1", s, []string{tpl.ActionID}, "relio-relevance-v1")
	requireFullCode(t, e, BundleLimitExceeded)
}
