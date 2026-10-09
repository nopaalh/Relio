package models

import "time"

// SourceSpan offsets count Unicode code points, with an exclusive End.
type SourceSpan struct {
	Start         int    `json:"start"`
	End           int    `json:"end"`
	FieldChecksum string `json:"field_checksum"`
}
type Evidence struct {
	EvidenceID        string            `json:"evidence_id"`
	SourceFile        string            `json:"source_file"`
	SourceRecordID    string            `json:"source_record_id"`
	RecordIDKind      string            `json:"record_id_kind"`
	SourceKey         map[string]string `json:"source_key"`
	SourceChecksum    string            `json:"source_checksum"`
	SourceField       string            `json:"source_field"`
	Span              *SourceSpan       `json:"span"`
	SourceDate        *Date             `json:"source_date"`
	SourceTimestamp   *time.Time        `json:"source_timestamp"`
	TemporalBasis     string            `json:"temporal_basis"`
	ContentExcerpt    string            `json:"content_excerpt"`
	ScopeKind         string            `json:"scope_kind"`
	VerificationState string            `json:"verification_state"`
	DealIDs           []string          `json:"deal_ids"`
	AccountIDs        []string          `json:"account_ids"`
	EventIDs          []string          `json:"event_ids"`
	EdgeIDs           []string          `json:"edge_ids"`
	OccurrenceIDs     []string          `json:"occurrence_ids"`
}
type EvidenceResult struct {
	Meta     Meta     `json:"meta"`
	Evidence Evidence `json:"evidence"`
}
