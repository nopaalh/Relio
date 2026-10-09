package models_test

import (
	"encoding/json"
	"testing"

	"github.com/nopaalh/Relio/backend/models"
)

// These are contract illustrations, not query results or competition records.
func TestDomainUnknownAndAmbiguousSerialization(t *testing.T) {
	person := models.ParticipantRef{
		VerificationState: "ambiguous", MatchMethod: "TEST-multiple-candidates",
		CandidateNodeIDs: []string{"TEST-PERSON-A", "TEST-PERSON-B"}, EvidenceIDs: []string{"TEST-EV"},
	}
	encoded, err := json.Marshal(person)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["node_id"]) != "null" {
		t.Fatalf("ambiguous identity chose node: %s", encoded)
	}
	if string(fields["verification_state"]) != `"ambiguous"` {
		t.Fatalf("identity changed verification: %s", encoded)
	}
	unknown := models.DealFacts{
		DealID: "TEST-DEAL", AccountID: "TEST-ACCOUNT",
		PotentialACVIDR:   models.Fact[int64]{State: "snapshot_only", TemporalBasis: "snapshot", EvidenceIDs: []string{}, Limitations: []string{"TEST-no-history"}},
		RecordEvidenceIDs: []string{},
	}
	encoded, err = json.Marshal(unknown)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	var acv map[string]json.RawMessage
	if err := json.Unmarshal(fields["potential_acv_idr"], &acv); err != nil {
		t.Fatal(err)
	}
	if string(acv["value"]) != "null" {
		t.Fatalf("historical ACV became zero: %s", encoded)
	}
}

func TestDomainEmptyResultArraysRemainArrays(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		fields []string
	}{
		{"deals", models.DealFactsPage{Items: []models.DealFacts{}}, []string{"items"}},
		{"graph", models.GraphResult{Nodes: []models.Node{}, Edges: []models.Edge{}}, []string{"nodes", "edges"}},
		{"timeline", models.TimelineResult{Events: []models.Event{}}, []string{"events"}},
		{"candidates", models.ActionCandidatesResult{Items: []models.ActionCandidate{}}, []string{"items"}},
		{"selected", models.SelectedActionsResult{SelectedActionIDs: []string{}, Items: []models.ActionCandidate{}, Evidence: []models.Evidence{}}, []string{"selected_action_ids", "items", "evidence"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			for _, key := range tt.fields {
				if string(fields[key]) != "[]" {
					t.Fatalf("%s = %s, want []", key, fields[key])
				}
			}
		})
	}
}
