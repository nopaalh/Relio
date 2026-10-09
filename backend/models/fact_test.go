package models_test

import (
	"encoding/json"
	"testing"

	"github.com/nopaalh/Relio/backend/models"
)

func TestFactNullZeroAndFalseRemainDistinct(t *testing.T) {
	zero, no := 0, false
	tests := []struct {
		name string
		fact any
		want string
	}{
		{"unknown", models.Fact[int]{State: "unknown", TemporalBasis: "event", EvidenceIDs: []string{}, Limitations: []string{}}, "null"},
		{"zero", models.Fact[int]{Value: &zero, State: "known", TemporalBasis: "event", EvidenceIDs: []string{"TEST-EV-ZERO"}, Limitations: []string{}}, "0"},
		{"false", models.Fact[bool]{Value: &no, State: "known", TemporalBasis: "event", EvidenceIDs: []string{"TEST-EV-FALSE"}, Limitations: []string{}}, "false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := json.Marshal(tt.fact)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			if string(fields["value"]) != tt.want {
				t.Fatalf("value = %s, want %s", fields["value"], tt.want)
			}
			if string(fields["limitations"]) != "[]" {
				t.Fatalf("limitations = %s", fields["limitations"])
			}
		})
	}
}

func TestFactKnownNeedsValueAndEvidence(t *testing.T) {
	zero := int64(0)
	tests := []struct {
		name     string
		value    *int64
		evidence []string
		valid    bool
	}{
		{"missing value", nil, []string{"TEST-EV"}, false},
		{"missing evidence", &zero, nil, false},
		{"blank evidence", &zero, []string{" "}, false},
		{"mixed blank reference", &zero, []string{"TEST-EV", ""}, false},
		{"known zero", &zero, []string{"TEST-EV"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fact := models.Fact[int64]{Value: tt.value, State: "known", TemporalBasis: "event", EvidenceIDs: tt.evidence}
			if err := fact.Validate(); (err == nil) != tt.valid {
				t.Fatalf("Validate = %v, valid=%v", err, tt.valid)
			}
		})
	}
}

func TestFactRejectsUnknownWithValue(t *testing.T) {
	value := "approved"
	for _, state := range []string{"unknown", "ambiguous", "snapshot_only"} {
		t.Run(state, func(t *testing.T) {
			fact := models.Fact[string]{State: state, TemporalBasis: "snapshot"}
			if err := fact.Validate(); err != nil {
				t.Fatal(err)
			}
			fact.Value = &value
			if err := fact.Validate(); err == nil {
				t.Fatal("unknown/ambiguous/snapshot-only selected a value")
			}
		})
	}
}

func TestFactRejectsInvalidStateAndBasis(t *testing.T) {
	for _, basis := range []string{"event", "interval", "snapshot", "undated"} {
		if err := (models.Fact[int]{State: "unknown", TemporalBasis: basis}).Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, fact := range []models.Fact[int]{
		{State: "", TemporalBasis: "event"},
		{State: "resolved", TemporalBasis: "event"},
		{State: "unknown", TemporalBasis: ""},
		{State: "unknown", TemporalBasis: "invented"},
	} {
		if err := fact.Validate(); err == nil {
			t.Fatalf("accepted invalid fact: %+v", fact)
		}
	}
}
