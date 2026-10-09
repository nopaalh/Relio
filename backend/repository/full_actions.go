package repository

import (
	"context"
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"reflect"
	"sort"
	"strings"
)

const FullRelevanceVersion = "relio-relevance-v1"

func fullGateEvidence(v visibleContext, account, actionType string) []string {
	words := map[string][]string{"discount": {"diskon", "harga", "budget", "bujet", "mahal"}, "starter_pilot": {"pilot", "uji coba"}, "escalation": {"eskalasi"}, "feature_promise": {"janji fitur"}}[actionType]
	ids := []string{}
	for _, e := range v.Evidence {
		if e.SourceFile != "interactions.jsonl" || e.SourceField != "isi" || len(e.AccountIDs) != 1 || e.AccountIDs[0] != account {
			continue
		}
		for _, w := range words {
			if strings.Contains(strings.ToLower(e.ContentExcerpt), w) {
				ids = append(ids, e.EvidenceID)
				break
			}
		}
	}
	sort.Strings(ids)
	return ids
}
func fullActionSet(raw fullSelection, v visibleContext, id string, s models.SnapshotContext) ([]models.ActionCandidate, error) {
	account := raw.Manifest.DealAccountIDs[id]
	expanded := s
	expanded.Access.AllowedAccountIDs = append(append([]string{}, s.Access.AllowedAccountIDs...), s.Access.AllowedAnalogAccountIDs...)
	expanded.Access.AllowedDealIDs = append([]string{}, s.Access.AllowedDealIDs...)
	for d, a := range raw.Manifest.DealAccountIDs {
		if s.Access.AllowsAnalogAccount(a) {
			expanded.Access.AllowedDealIDs = append(expanded.Access.AllowedDealIDs, d)
		}
	}
	proofs, err := projectFull(raw, expanded)
	if err != nil {
		return nil, err
	}
	sets := map[string]map[string]bool{"EvidenceIDs": {}, "NodeIDs": {}, "CandidateNodeIDs": {}, "EdgeIDs": {}, "EventIDs": {}, "AccountIDs": {}, "DealIDs": {}}
	for _, e := range proofs.Evidence {
		sets["EvidenceIDs"][e.EvidenceID] = true
	}
	for _, n := range proofs.Nodes {
		sets["NodeIDs"][n.NodeID] = true
		sets["CandidateNodeIDs"][n.NodeID] = true
	}
	for _, e := range proofs.Edges {
		sets["EdgeIDs"][e.EdgeID] = true
	}
	for _, e := range proofs.Events {
		sets["EventIDs"][e.EventID] = true
	}
	for _, a := range expanded.Access.AllowedAccountIDs {
		sets["AccountIDs"][a] = true
	}
	for _, d := range expanded.Access.AllowedDealIDs {
		sets["DealIDs"][d] = true
	}
	templates := map[string]models.ActionTemplate{}
	occurrences := []models.ActionOccurrence{}
	for _, r := range raw.Records {
		if r.AvailableFrom > s.AsOf || !fullRecordAllowed(r, s, true) {
			continue
		}
		if r.Kind == "templates" {
			var t models.ActionTemplate
			_ = json.Unmarshal(r.Payload, &t)
			if t.TaxonomyVersion == "relio-actions-v1" {
				templates[t.ActionID] = t
			}
		}
		if r.Kind == "occurrences" {
			var o models.ActionOccurrence
			_ = json.Unmarshal(r.Payload, &o)
			if o.EventAt > s.AsOf || !sets["EventIDs"][o.EventID] {
				continue
			}
			pruneProjection(reflect.ValueOf(&o).Elem(), s.AsOf, sets)
			clearUnprovedParticipants(reflect.ValueOf(&o).Elem())
			if len(o.EvidenceIDs) == 0 || o.Status.Value == nil {
				continue
			}
			if s.AsOf < MaxAsOf {
				o.Policy.AppliedContractID = nil
			}
			if o.OutcomeAt != nil && *o.OutcomeAt > s.AsOf {
				o.OutcomeAt = nil
				o.OutcomeObserved.Value = nil
				o.OutcomeObserved.State = "unknown"
			}
			occurrences = append(occurrences, o)
		}
	}
	sort.Slice(occurrences, func(i, j int) bool { return occurrences[i].OccurrenceID < occurrences[j].OccurrenceID })
	items := []models.ActionCandidate{}
	for _, t := range templates {
		gates := fullGateEvidence(v, account, t.ActionType)
		if len(gates) == 0 {
			continue
		}
		c := models.ActionCandidate{Template: t, Precedents: []models.ActionPrecedent{}, RelevanceReasons: []models.RelevanceReason{}, Limitations: []string{"precedent_not_current_authorization", "reference_consent_unknown", "approver_historical_role_unknown", "deterministic_keyword_relevance_not_suitability"}}
		for _, o := range occurrences {
			if o.ActionID != t.ActionID || len(o.AccountIDs) != 1 {
				continue
			}
			a := o.AccountIDs[0]
			relation := ""
			if a == account {
				if len(o.DealIDs) == 0 {
					relation = "account_context"
				} else {
					for _, d := range o.DealIDs {
						if d == id {
							relation = "current_deal"
						}
					}
				}
			} else if s.Access.AllowsAnalogAccount(a) {
				relation = "analog_precedent"
			}
			if relation == "" {
				continue
			}
			c.Precedents = append(c.Precedents, models.ActionPrecedent{Relation: relation, Occurrence: o})
			c.RelevanceReasons = append(c.RelevanceReasons, models.RelevanceReason{Code: "explicit_account_request_keyword", Detail: "Source text matches the versioned request/gate rule; suitability belongs to O2.", DealEvidenceIDs: gates, PrecedentEvidenceIDs: o.EvidenceIDs})
		}
		if len(c.Precedents) > 0 {
			items = append(items, c)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Template.ActionID < items[j].Template.ActionID })
	return items, nil
}
func (r *Neo4jFullContextRepository) ReadActionCandidates(ctx context.Context, id string, s models.SnapshotContext, o models.CandidateOptions) (models.ActionCandidatesResult, error) {
	if o.RelevanceRuleVersion != FullRelevanceVersion {
		return models.ActionCandidatesResult{}, &RepositoryError{Code: InvalidQuery}
	}
	limit, e := NormalizePageLimit(o.Limit)
	if e != nil {
		return models.ActionCandidatesResult{}, e
	}
	key := filterKey("actions", id, o.RelevanceRuleVersion, limit)
	after, e := cursorAfter(o.Cursor, s.ContextID, key)
	if e != nil {
		return models.ActionCandidatesResult{}, e
	}
	raw, v, e := r.loadFull(ctx, s, fullQueryFor(s, "actions"))
	if e != nil {
		return models.ActionCandidatesResult{}, e
	}
	if _, e = fullPrimaryDeal(raw, v, id, s); e != nil {
		return models.ActionCandidatesResult{}, e
	}
	v, e = fullViewForDeal(raw, s, id)
	if e != nil {
		return models.ActionCandidatesResult{}, e
	}
	all, e := fullActionSet(raw, v, id, s)
	if e != nil {
		return models.ActionCandidatesResult{}, e
	}
	total := len(all)
	out := models.ActionCandidatesResult{Meta: v.Meta, Bounds: emptyBounds(), RelevanceRuleVersion: FullRelevanceVersion, TotalValidCount: &total, Items: []models.ActionCandidate{}}
	found := after == ""
	for _, c := range all {
		if c.Template.ActionID == after {
			found = true
		}
		if c.Template.ActionID > after {
			out.Items = append(out.Items, c)
		}
	}
	if !found {
		return out, &RepositoryError{Code: InvalidQuery}
	}
	if len(out.Items) > limit {
		out.Items = out.Items[:limit]
		out.Bounds.Truncated = true
		out.Bounds.TruncationReasons = []string{"page_limit"}
		out.Bounds.NextCursor = nextCursor(s.ContextID, key, out.Items[len(out.Items)-1].Template.ActionID)
	}
	out.ReturnedCount = len(out.Items)
	return out, nil
}
func (r *Neo4jFullContextRepository) ReadSelectedActions(ctx context.Context, id string, s models.SnapshotContext, ids []string, version string) (models.SelectedActionsResult, error) {
	if version != FullRelevanceVersion {
		return models.SelectedActionsResult{}, &RepositoryError{Code: InvalidQuery}
	}
	unique, e := canonicalIDs(ids)
	if e != nil || len(unique) != len(ids) || len(ids) < 1 || len(ids) > 4 {
		return models.SelectedActionsResult{}, &RepositoryError{Code: InvalidSelection}
	}
	raw, v, e := r.loadFull(ctx, s, fullQueryFor(s, "actions"))
	if e != nil {
		return models.SelectedActionsResult{}, e
	}
	if _, e = fullPrimaryDeal(raw, v, id, s); e != nil {
		return models.SelectedActionsResult{}, e
	}
	v, e = fullViewForDeal(raw, s, id)
	if e != nil {
		return models.SelectedActionsResult{}, e
	}
	all, e := fullActionSet(raw, v, id, s)
	if e != nil {
		return models.SelectedActionsResult{}, e
	}
	out := models.SelectedActionsResult{Meta: v.Meta, RelevanceRuleVersion: version, SelectedActionIDs: append([]string{}, ids...), Items: []models.ActionCandidate{}, Evidence: []models.Evidence{}}
	proofIDs := map[string]bool{}
	for _, wanted := range ids {
		found := false
		for _, c := range all {
			if c.Template.ActionID != wanted {
				continue
			}
			found = true
			out.Items = append(out.Items, c)
			for _, p := range c.Precedents {
				for _, eid := range p.Occurrence.EvidenceIDs {
					proofIDs[eid] = true
				}
			}
			for _, reason := range c.RelevanceReasons {
				for _, eid := range reason.DealEvidenceIDs {
					proofIDs[eid] = true
				}
				for _, eid := range reason.PrecedentEvidenceIDs {
					proofIDs[eid] = true
				}
			}
		}
		if !found {
			return models.SelectedActionsResult{}, &RepositoryError{Code: InvalidSelection}
		}
	}
	// Return exactly supporting fields, not threads or unrelated account records.
	expanded := s
	expanded.Access.AllowedAccountIDs = append(append([]string{}, s.Access.AllowedAccountIDs...), s.Access.AllowedAnalogAccountIDs...)
	expanded.Access.AllowedDealIDs = append([]string{}, s.Access.AllowedDealIDs...)
	for d, a := range raw.Manifest.DealAccountIDs {
		if s.Access.AllowsAnalogAccount(a) {
			expanded.Access.AllowedDealIDs = append(expanded.Access.AllowedDealIDs, d)
		}
	}
	proofs, e := projectFull(raw, expanded)
	if e != nil {
		return models.SelectedActionsResult{}, e
	}
	for _, ev := range proofs.Evidence {
		if proofIDs[ev.EvidenceID] {
			out.Evidence = append(out.Evidence, ev)
			delete(proofIDs, ev.EvidenceID)
		}
	}
	if len(proofIDs) > 0 {
		return models.SelectedActionsResult{}, &RepositoryError{Code: InvalidSelection}
	}
	sort.Slice(out.Evidence, func(i, j int) bool { return out.Evidence[i].EvidenceID < out.Evidence[j].EvidenceID })
	if r.MaxSelectedEvidenceBytes > 0 {
		b, _ := json.Marshal(out)
		if len(b) > r.MaxSelectedEvidenceBytes {
			return models.SelectedActionsResult{}, &RepositoryError{Code: BundleLimitExceeded}
		}
	}
	return out, nil
}

var _ ActionRepository = (*Neo4jFullContextRepository)(nil)
