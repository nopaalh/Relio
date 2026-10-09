package services

import (
	"context"
	"sync"

	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
)

type ContextRepository interface {
	repository.DealFactsRepository
	repository.GraphRepository
	repository.TimelineRepository
	repository.EvidenceRepository
}

type JEVEvaluator interface {
	Evaluate(context.Context, models.JEVRequest) (models.JEVResponse, error)
}

type ContextService struct {
	reader         ContextRepository
	datasetVersion string
	access         models.AccessScope
	jev            JEVEvaluator
	assessmentMu   sync.Mutex
	assessments    map[string]cachedAssessment
}

func NewContextService(reader ContextRepository, datasetVersion string, access models.AccessScope, jev JEVEvaluator) *ContextService {
	access.AllowedAccountIDs = append([]string{}, access.AllowedAccountIDs...)
	access.AllowedDealIDs = append([]string{}, access.AllowedDealIDs...)
	access.AllowedAnalogAccountIDs = append([]string{}, access.AllowedAnalogAccountIDs...)
	return &ContextService{reader: reader, datasetVersion: datasetVersion, access: access, jev: jev, assessments: make(map[string]cachedAssessment)}
}

func (s *ContextService) Snapshot(asOf models.Date) (models.SnapshotContext, error) {
	snapshot, err := repository.NewSnapshotContext(asOf, s.datasetVersion, s.access)
	if err != nil {
		return models.SnapshotContext{}, err
	}
	if s.reader == nil {
		return models.SnapshotContext{}, &repository.RepositoryError{Code: repository.DataNotReady}
	}
	return snapshot, nil
}

func (s *ContextService) validate(ctx context.Context, snapshot models.SnapshotContext) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	expected, err := s.Snapshot(snapshot.AsOf)
	if err != nil {
		return err
	}
	if err := repository.ValidateSnapshot(snapshot); err != nil {
		return err
	}
	if expected.ContextID != snapshot.ContextID {
		return &repository.RepositoryError{Code: repository.ContextMismatch}
	}
	return nil
}

func contextMeta(snapshot models.SnapshotContext, meta models.Meta) error {
	if meta.AsOf != snapshot.AsOf || meta.DatasetVersion != snapshot.DatasetVersion || meta.ContractVersion != snapshot.ContractVersion || meta.CalendarZone != snapshot.CalendarZone || meta.ContextID != snapshot.ContextID || meta.MaxAsOf != repository.MaxAsOf || meta.DataState == "" {
		return &repository.RepositoryError{Code: repository.ContextMismatch}
	}
	return nil
}

func (s *ContextService) List(ctx context.Context, snapshot models.SnapshotContext, options models.DealListOptions) (models.DealFactsPage, error) {
	if err := s.validate(ctx, snapshot); err != nil {
		return models.DealFactsPage{}, err
	}
	result, err := s.reader.ListFacts(ctx, snapshot, options)
	if err == nil {
		err = contextMeta(snapshot, result.Meta)
	}
	if err != nil {
		return models.DealFactsPage{}, err
	}
	return models.NormalizeDealPage(result), nil
}

func (s *ContextService) Deal(ctx context.Context, id string, snapshot models.SnapshotContext) (models.DealFactsResult, error) {
	if err := s.validate(ctx, snapshot); err != nil {
		return models.DealFactsResult{}, err
	}
	result, err := s.reader.FindFacts(ctx, id, snapshot)
	if err == nil {
		err = contextMeta(snapshot, result.Meta)
	}
	if err != nil {
		return models.DealFactsResult{}, err
	}
	return models.NormalizeDealResult(result), nil
}

func (s *ContextService) Graph(ctx context.Context, id string, snapshot models.SnapshotContext, options models.GraphOptions) (models.GraphResult, error) {
	if err := s.validate(ctx, snapshot); err != nil {
		return models.GraphResult{}, err
	}
	result, err := s.reader.ReadGraph(ctx, id, snapshot, options)
	if err == nil {
		err = contextMeta(snapshot, result.Meta)
	}
	if err != nil {
		return models.GraphResult{}, err
	}
	return models.NormalizeGraphResult(result), nil
}

func (s *ContextService) Timeline(ctx context.Context, id string, snapshot models.SnapshotContext, options models.TimelineOptions) (models.TimelineResult, error) {
	if err := s.validate(ctx, snapshot); err != nil {
		return models.TimelineResult{}, err
	}
	result, err := s.reader.ReadTimeline(ctx, id, snapshot, options)
	if err == nil {
		err = contextMeta(snapshot, result.Meta)
	}
	if err != nil {
		return models.TimelineResult{}, err
	}
	return models.NormalizeTimelineResult(result), nil
}

func (s *ContextService) Evidence(ctx context.Context, id string, snapshot models.SnapshotContext) (models.EvidenceResult, error) {
	if err := s.validate(ctx, snapshot); err != nil {
		return models.EvidenceResult{}, err
	}
	result, err := s.reader.ReadEvidence(ctx, id, snapshot)
	if err == nil {
		err = contextMeta(snapshot, result.Meta)
	}
	if err != nil {
		return models.EvidenceResult{}, err
	}
	return models.NormalizeEvidenceResult(result), nil
}

func (s *ContextService) ActionCandidates(ctx context.Context, id string, snapshot models.SnapshotContext, options models.CandidateOptions) (models.ActionCandidatesResult, error) {
	if err := s.validate(ctx, snapshot); err != nil {
		return models.ActionCandidatesResult{}, err
	}
	actions, ok := s.reader.(repository.ActionRepository)
	if !ok {
		return models.ActionCandidatesResult{}, &repository.RepositoryError{Code: repository.DataNotReady}
	}
	result, err := actions.ReadActionCandidates(ctx, id, snapshot, options)
	if err == nil {
		err = contextMeta(snapshot, result.Meta)
	}
	if err != nil {
		return models.ActionCandidatesResult{}, err
	}
	return result, nil
}

func (s *ContextService) CompareActions(ctx context.Context, id string, snapshot models.SnapshotContext, ids []string, version string) (models.SelectedActionsResult, error) {
	if err := s.validate(ctx, snapshot); err != nil {
		return models.SelectedActionsResult{}, err
	}
	actions, ok := s.reader.(repository.ActionRepository)
	if !ok {
		return models.SelectedActionsResult{}, &repository.RepositoryError{Code: repository.DataNotReady}
	}
	result, err := actions.ReadSelectedActions(ctx, id, snapshot, ids, version)
	if err == nil {
		err = contextMeta(snapshot, result.Meta)
	}
	if err != nil {
		return models.SelectedActionsResult{}, err
	}
	return result, nil
}
