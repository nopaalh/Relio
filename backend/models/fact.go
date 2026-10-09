package models

import (
	"errors"
	"strings"
)

// Fact preserves absence, ambiguity, and provenance. It is not an HTTP DTO.
type Fact[T any] struct {
	Value         *T       `json:"value"`
	State         string   `json:"state"`
	TemporalBasis string   `json:"temporal_basis"`
	EvidenceIDs   []string `json:"evidence_ids"`
	Limitations   []string `json:"limitations"`
}

// Validate checks scalar-state consistency, not evidence existence, access,
// historical visibility, or domain-specific values. Adapters check those.
func (f Fact[T]) Validate() error {
	switch f.TemporalBasis {
	case "event", "interval", "snapshot", "undated":
	default:
		return errors.New("invalid fact temporal basis")
	}
	for _, id := range f.EvidenceIDs {
		if strings.TrimSpace(id) == "" {
			return errors.New("empty fact evidence reference")
		}
	}
	switch f.State {
	case "known":
		if f.Value == nil || len(f.EvidenceIDs) == 0 {
			return errors.New("known fact requires value and evidence")
		}
	case "unknown", "ambiguous", "snapshot_only":
		if f.Value != nil {
			return errors.New("unresolved fact must not select a value")
		}
	default:
		return errors.New("invalid fact state")
	}
	return nil
}
