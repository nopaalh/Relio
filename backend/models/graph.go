package models

import "time"

type ParticipantRef struct {
	NodeID            *string  `json:"node_id"`
	RawRef            *string  `json:"raw_ref"`
	VerificationState string   `json:"verification_state"`
	MatchMethod       string   `json:"match_method"`
	CandidateNodeIDs  []string `json:"candidate_node_ids"`
	EvidenceIDs       []string `json:"evidence_ids"`
}
type Validity struct {
	ValidFrom     *Date   `json:"valid_from"`
	ValidTo       *Date   `json:"valid_to"`
	OpenEndReason *string `json:"open_end_reason"`
	TemporalBasis string  `json:"temporal_basis"`
}
type Node struct {
	NodeID            string          `json:"node_id"`
	NodeType          string          `json:"node_type"`
	EntityID          string          `json:"entity_id"`
	PersonKind        *string         `json:"person_kind"`
	Person            *ParticipantRef `json:"person"`
	Label             Fact[string]    `json:"label"`
	Validity          Validity        `json:"validity"`
	VerificationState string          `json:"verification_state"`
	EvidenceIDs       []string        `json:"evidence_ids"`
	EventIDs          []string        `json:"event_ids"`
}
type Edge struct {
	EdgeID            string     `json:"edge_id"`
	EdgeType          string     `json:"edge_type"`
	Source            string     `json:"source"`
	Target            string     `json:"target"`
	Validity          Validity   `json:"validity"`
	SourceTimestamp   *time.Time `json:"source_timestamp"`
	ObservedAt        *time.Time `json:"observed_at"`
	RecordedAt        *time.Time `json:"recorded_at"`
	VerificationState string     `json:"verification_state"`
	MatchMethod       string     `json:"match_method"`
	EvidenceIDs       []string   `json:"evidence_ids"`
	EventIDs          []string   `json:"event_ids"`
}
type GraphOptions struct {
	Depth        int
	MaxNodes     int
	MaxEdges     int
	FocusEventID *string
}
type GraphResult struct {
	Meta   Meta      `json:"meta"`
	Bounds BoundInfo `json:"bounds"`
	Nodes  []Node    `json:"nodes"`
	Edges  []Edge    `json:"edges"`
}
