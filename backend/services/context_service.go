package services

import (
	"context"
	"fmt"
	"math"
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

func (s *ContextService) assessmentEvidence(ctx context.Context, ids []string, snapshot models.SnapshotContext) ([]models.EvidenceResult, error) {
	if err := s.validate(ctx, snapshot); err != nil {
		return nil, err
	}
	if batch, ok := s.reader.(repository.EvidenceBatchRepository); ok {
		results, err := batch.ReadEvidenceBatch(ctx, ids, snapshot)
		if err != nil {
			return nil, err
		}
		if len(results) != len(ids) {
			return nil, &repository.RepositoryError{Code: repository.ContextMismatch}
		}
		for i := range results {
			if err := contextMeta(snapshot, results[i].Meta); err != nil {
				return nil, err
			}
			results[i] = models.NormalizeEvidenceResult(results[i])
		}
		return results, nil
	}
	return readAssessmentEvidence(ctx, ids, func(ctx context.Context, id string) (models.EvidenceResult, error) {
		return s.Evidence(ctx, id, snapshot)
	})
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

func (s *ContextService) CompareActions(ctx context.Context, id string, snapshot models.SnapshotContext, ids []string, version string) (models.ActionComparisonResult, error) {
	if err := s.validate(ctx, snapshot); err != nil {
		return models.ActionComparisonResult{}, err
	}
	actions, ok := s.reader.(repository.ActionRepository)
	if !ok {
		return models.ActionComparisonResult{}, &repository.RepositoryError{Code: repository.DataNotReady}
	}
	selected, err := actions.ReadSelectedActions(ctx, id, snapshot, ids, version)
	if err == nil {
		err = contextMeta(snapshot, selected.Meta)
	}
	if err != nil {
		return models.ActionComparisonResult{}, err
	}
	result := models.ActionComparisonResult{Meta: selected.Meta, DealID: id, AsOf: snapshot.AsOf, ComparisonType: "historical_precedent_suitability", AssessmentStatus: "jev_unavailable", RubricVersion: "action-suitability-v1.0", Items: []models.ActionComparisonItem{}, Warning: "Historical precedent is not approval or consent for this deal. Score is suitability, not closing probability."}
	questions := make(map[string]models.JEVQuestion, len(selected.Items)+1)
	options := make(map[string]any, len(selected.Items))
	keys := make([]string, len(selected.Items))
	for i, candidate := range selected.Items {
		key := fmt.Sprintf("suitability_%d", i)
		keys[i] = key
		options[candidate.Template.ActionID] = candidate.Template.Definition
		item := models.ActionComparisonItem{ActionID: candidate.Template.ActionID, EvidenceIDs: []string{}, PrecedentIDs: []string{}, PolicyFlags: []string{}, Unknowns: append([]string{}, candidate.Limitations...)}
		for _, precedent := range candidate.Precedents {
			item.PrecedentIDs = append(item.PrecedentIDs, precedent.Occurrence.OccurrenceID)
			item.EvidenceIDs = append(item.EvidenceIDs, precedent.Occurrence.EvidenceIDs...)
		}
		result.Items = append(result.Items, item)
		questions[key] = models.JEVQuestion{Type: models.JEVQuestionTypeScore, Instructions: "Rate suitability of this historical approach for the current deal using only supplied evidence. Historical use, approval, outcome, or consent does not authorize a new action. If evidence is insufficient, do not infer a high score.", ScoreCriteria: []any{"0: contradicted or unsuitable", "1: weak fit", "2: plausible with substantial validation", "3: good evidence-backed fit, still requires human validation", "4: strong evidence-backed fit, still requires human validation"}}
	}
	if len(selected.Items) >= 2 {
		questions["relative_preference"] = models.JEVQuestion{Type: models.JEVQuestionTypeChoice, Instructions: "Choose the most relevant option for this deal from the selected historical approaches. This is relative preference only, not probability, authorization, or consent.", ChoiceCriteria: options}
	}
	if s.jev == nil || len(selected.Items) < 2 {
		for i := range result.Items {
			result.Items[i].Unknowns = append(result.Items[i].Unknowns, "JEV comparison unavailable; no suitability score, ranking, or preference was produced.")
		}
		if len(selected.Items) < 2 {
			result.AssessmentStatus = "insufficient_evidence"
			result.Warning += " Fewer than two valid sourced candidates are available."
		}
		return result, nil
	}
	deal, err := s.Deal(ctx, id, snapshot)
	if err != nil {
		return models.ActionComparisonResult{}, err
	}
	req := models.JEVRequest{Model: models.JEVModelLatest, State: map[string]any{"as_of": snapshot.AsOf, "deal": deal.Deal, "selected_actions": selected.Items, "evidence": selected.Evidence}, Questions: questions}
	response, err := s.jev.Evaluate(ctx, req)
	if err == nil {
		err = repository.ValidateJEVResponse(req.Questions, response)
	}
	if err != nil {
		for i := range result.Items {
			result.Items[i].Unknowns = append(result.Items[i].Unknowns, "JEV unavailable or invalid; no score, ranking, or preference was produced.")
		}
		return result, nil
	}
	result.AssessmentStatus = "assessed"
	for i, item := range result.Items {
		answer := response.Answers[keys[i]]
		if answer.Score != nil {
			raw := *answer.Score
			score := int(math.Round(raw * 25))
			item.JEVRawScore = &raw
			item.SuitabilityScore100 = &score
			item.JEVConfidence = answer.Confidence
		}
		if pref, ok := response.Answers["relative_preference"].Probabilities[item.ActionID]; ok {
			value := pref
			item.JEVChoicePreference = &value
		}
		if answer.Score == nil {
			item.Unknowns = append(item.Unknowns, "JEV did not provide a score for this option.")
		}
	}
	allScored := true
	for _, item := range result.Items {
		if item.SuitabilityScore100 == nil {
			allScored = false
		}
	}
	if allScored {
		for i := range result.Items {
			rank := 1
			for _, other := range result.Items {
				if *other.SuitabilityScore100 > *result.Items[i].SuitabilityScore100 {
					rank++
				}
			}
			result.Items[i].Rank = &rank
		}
	}
	return result, nil
}
