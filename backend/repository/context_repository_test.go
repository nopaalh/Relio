package repository_test

import (
	"context"
	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
	"time"
)

// Method-set proof only. This test type is not a runtime data adapter.
type contractStub struct{}

var (
	_ repository.DealFactsRepository = (*contractStub)(nil)
	_ repository.GraphRepository     = (*contractStub)(nil)
	_ repository.TimelineRepository  = (*contractStub)(nil)
	_ repository.EvidenceRepository  = (*contractStub)(nil)
	_ repository.ActionRepository    = (*contractStub)(nil)
	// The rich contract must not remove or change the legacy method set.
	_ interface {
		List(context.Context, time.Time) ([]models.Deal, error)
		FindByID(context.Context, string, time.Time) (models.Deal, error)
	} = (repository.DealRepository)(nil)
)

func (*contractStub) ListFacts(context.Context, models.SnapshotContext, models.DealListOptions) (models.DealFactsPage, error) {
	return models.DealFactsPage{}, &repository.RepositoryError{Code: repository.DataNotReady}
}
func (*contractStub) FindFacts(context.Context, string, models.SnapshotContext) (models.DealFactsResult, error) {
	return models.DealFactsResult{}, &repository.RepositoryError{Code: repository.DataNotReady}
}
func (*contractStub) ReadGraph(context.Context, string, models.SnapshotContext, models.GraphOptions) (models.GraphResult, error) {
	return models.GraphResult{}, &repository.RepositoryError{Code: repository.DataNotReady}
}
func (*contractStub) ReadTimeline(context.Context, string, models.SnapshotContext, models.TimelineOptions) (models.TimelineResult, error) {
	return models.TimelineResult{}, &repository.RepositoryError{Code: repository.DataNotReady}
}
func (*contractStub) ReadEvidence(context.Context, string, models.SnapshotContext) (models.EvidenceResult, error) {
	return models.EvidenceResult{}, &repository.RepositoryError{Code: repository.DataNotReady}
}
func (*contractStub) ReadActionCandidates(context.Context, string, models.SnapshotContext, models.CandidateOptions) (models.ActionCandidatesResult, error) {
	return models.ActionCandidatesResult{}, &repository.RepositoryError{Code: repository.DataNotReady}
}
func (*contractStub) ReadSelectedActions(context.Context, string, models.SnapshotContext, []string, string) (models.SelectedActionsResult, error) {
	return models.SelectedActionsResult{}, &repository.RepositoryError{Code: repository.DataNotReady}
}
