package repository

import (
	"context"
	"github.com/nopaalh/Relio/backend/models"
)

// Candidate shared contract for O2 review. All operations are read-only.
// Adapters enforce snapshot, server access scope, and reference visibility.
// The legacy DealRepository remains separate; no lossy compatibility bridge.
type DealFactsRepository interface {
	ListFacts(ctx context.Context, snapshot models.SnapshotContext, options models.DealListOptions) (models.DealFactsPage, error)
	FindFacts(ctx context.Context, dealID string, snapshot models.SnapshotContext) (models.DealFactsResult, error)
}
type GraphRepository interface {
	ReadGraph(ctx context.Context, dealID string, snapshot models.SnapshotContext, options models.GraphOptions) (models.GraphResult, error)
}
type TimelineRepository interface {
	ReadTimeline(ctx context.Context, dealID string, snapshot models.SnapshotContext, options models.TimelineOptions) (models.TimelineResult, error)
}
type EvidenceRepository interface {
	ReadEvidence(ctx context.Context, evidenceID string, snapshot models.SnapshotContext) (models.EvidenceResult, error)
}
type EvidenceBatchRepository interface {
	ReadEvidenceBatch(ctx context.Context, evidenceIDs []string, snapshot models.SnapshotContext) ([]models.EvidenceResult, error)
}
type ActionRepository interface {
	ReadActionCandidates(ctx context.Context, dealID string, snapshot models.SnapshotContext, options models.CandidateOptions) (models.ActionCandidatesResult, error)
	// Resolve every selected ID against the same scope/snapshot/relevance version.
	// Return a complete evidence bundle or an atomic error, never a partial success.
	// One ID may be inspected here; O2 enforces 2–4 unique IDs for comparison.
	ReadSelectedActions(ctx context.Context, dealID string, snapshot models.SnapshotContext, selectedTemplateIDs []string, relevanceRuleVersion string) (models.SelectedActionsResult, error)
}
