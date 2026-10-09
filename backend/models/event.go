package models

import "time"

type Event struct {
	EventID           string           `json:"event_id"`
	VerificationState string           `json:"verification_state"`
	EventAt           Date             `json:"event_at"`
	TimePrecision     string           `json:"time_precision"`
	SourceTimestamp   *time.Time       `json:"source_timestamp"`
	EventType         string           `json:"event_type"`
	Status            Fact[string]     `json:"status"`
	RawStatus         *string          `json:"raw_status"`
	Summary           Fact[string]     `json:"summary"`
	Actors            []ParticipantRef `json:"actors"`
	Targets           []ParticipantRef `json:"targets"`
	Participants      []ParticipantRef `json:"participants"`
	ScopeKind         string           `json:"scope_kind"`
	DealIDs           []string         `json:"deal_ids"`
	AccountIDs        []string         `json:"account_ids"`
	NodeIDs           []string         `json:"node_ids"`
	EdgeIDs           []string         `json:"edge_ids"`
	EvidenceIDs       []string         `json:"evidence_ids"`
}
type TimelineOptions struct {
	EventIDs     []string
	EventTypes   []string
	ActorNodeIDs []string
	Statuses     []string
	Limit        int
	Cursor       *string
}
type TimelineResult struct {
	Meta   Meta      `json:"meta"`
	Bounds BoundInfo `json:"bounds"`
	Events []Event   `json:"events"`
}
