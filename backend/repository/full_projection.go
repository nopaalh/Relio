package repository

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/nopaalh/Relio/backend/models"
	"reflect"
)

func fullRecordAllowed(r FullRecord, s models.SnapshotContext, analogs bool) bool {
	if len(r.AccountIDs) == 0 {
		// An external contact's unassigned employment is not company-internal evidence.
		if r.Kind == "evidence" {
			var source struct {
				SourceFile string `json:"source_file"`
			}
			if json.Unmarshal(r.Payload, &source) != nil {
				return false
			}
			if source.SourceFile == "contact_employment_history.csv" {
				return false
			}
		}
		return s.Access.AllowCompanyEvidence && len(r.DealIDs) == 0
	}
	for _, a := range r.AccountIDs {
		if analogs && s.Access.AllowsAnalogAccount(a) {
			return true
		}
		if !s.Access.AllowsAccount(a) {
			continue
		}
		if len(r.DealIDs) == 0 {
			return true
		}
		for _, d := range r.DealIDs {
			if s.Access.AllowsDeal(d, a) {
				return true
			}
		}
	}
	return false
}
func fullMeta(s models.SnapshotContext) models.Meta {
	return models.Meta{ContractVersion: s.ContractVersion, DatasetVersion: s.DatasetVersion, AsOf: s.AsOf, MaxAsOf: MaxAsOf, CalendarZone: s.CalendarZone, ContextID: s.ContextID, DataState: "ready", Limitations: []string{"snapshot_fields_have_no_invented_history", "historical_email_aliases_unresolved", "account_events_are_not_native_deal_events"}, Unknowns: []models.Unknown{}}
}
func (r *Neo4jFullContextRepository) load(ctx context.Context, s models.SnapshotContext) (fullSelection, visibleContext, error) {
	return r.loadFull(ctx, s, fullQueryFor(s, "context"))
}
func (r *Neo4jFullContextRepository) loadFull(ctx context.Context, s models.SnapshotContext, q fullQuery) (fullSelection, visibleContext, error) {
	var raw fullSelection
	var v visibleContext
	if e := ValidateSnapshot(s); e != nil {
		return raw, v, e
	}
	if e := ctx.Err(); e != nil {
		return raw, v, &RepositoryError{Code: QueryFailed, Cause: e}
	}
	if len(s.Access.AllowedAccountIDs) == 0 {
		return raw, v, &RepositoryError{Code: AccessDenied}
	}
	if r == nil || r.read == nil {
		return raw, v, &RepositoryError{Code: DataNotReady}
	}
	raw, e := r.read(ctx, q)
	if e != nil {
		var re *RepositoryError
		if errors.As(e, &re) {
			return raw, v, e
		}
		return raw, v, &RepositoryError{Code: QueryFailed, Cause: e}
	}
	if raw.Manifest.DatasetVersion != s.DatasetVersion {
		return raw, v, &RepositoryError{Code: DataNotReady}
	}
	if e = verifyFullSelection(raw.Manifest, raw.Partitions, raw.Records); e != nil {
		return raw, v, e
	}
	v, e = projectFull(raw, s)
	return raw, v, e
}
func projectFull(raw fullSelection, s models.SnapshotContext) (visibleContext, error) {
	v := visibleContext{Meta: fullMeta(s), Deals: []models.DealFacts{}, Nodes: []models.Node{}, Edges: []models.Edge{}, Events: []models.Event{}, Evidence: []models.Evidence{}}
	for _, r := range raw.Records {
		if r.AvailableFrom > s.AsOf || !fullRecordAllowed(r, s, false) {
			continue
		}
		var out any
		switch r.Kind {
		case "deals":
			out = &models.DealFacts{}
		case "nodes":
			out = &models.Node{}
		case "edges":
			out = &models.Edge{}
		case "events":
			out = &models.Event{}
		case "evidence":
			out = &models.Evidence{}
		default:
			continue
		}
		if e := json.Unmarshal(r.Payload, out); e != nil {
			return v, &RepositoryError{Code: DataNotReady, Cause: e}
		}
		switch x := out.(type) {
		case *models.DealFacts:
			if s.Access.AllowsDeal(x.DealID, x.AccountID) {
				v.Deals = append(v.Deals, *x)
			}
		case *models.Node:
			if validAt(x.Validity, s.AsOf) {
				if x.NodeType == "deal" && !s.Access.AllowsDeal(x.EntityID, raw.Manifest.DealAccountIDs[x.EntityID]) {
					continue
				}
				maskValidity(&x.Validity, s.AsOf)
				v.Nodes = append(v.Nodes, *x)
			}
		case *models.Edge:
			if validAt(x.Validity, s.AsOf) {
				maskValidity(&x.Validity, s.AsOf)
				v.Edges = append(v.Edges, *x)
			}
		case *models.Event:
			if x.EventAt <= s.AsOf {
				v.Events = append(v.Events, *x)
			}
		case *models.Evidence:
			if x.SourceDate == nil || *x.SourceDate <= s.AsOf {
				if x.SourceFile == "contact_employment_history.csv" && x.SourceKey["selesai"] != "" && models.Date(x.SourceKey["selesai"]) > s.AsOf {
					delete(x.SourceKey, "selesai")
					v.Meta.Limitations = append(v.Meta.Limitations, "composite_locator_future_end_redacted_use_evidence_id_and_source_checksum")
				}
				v.Evidence = append(v.Evidence, *x)
			}
		}
	}
	sets := map[string]map[string]bool{"EvidenceIDs": {}, "RecordEvidenceIDs": {}, "NodeIDs": {}, "CandidateNodeIDs": {}, "EdgeIDs": {}, "EventIDs": {}, "DealIDs": {}, "AccountIDs": {}, "OccurrenceIDs": {}}
	for _, a := range s.Access.AllowedAccountIDs {
		sets["AccountIDs"][a] = true
	}
	for _, e := range v.Evidence {
		sets["EvidenceIDs"][e.EvidenceID] = true
		sets["RecordEvidenceIDs"][e.EvidenceID] = true
	}
	// A shared person's presence cannot make another account's identity proof visible.
	nodes := []models.Node{}
	for _, n := range v.Nodes {
		hit := false
		for _, id := range n.EvidenceIDs {
			if sets["EvidenceIDs"][id] {
				hit = true
			}
		}
		if hit {
			nodes = append(nodes, n)
			sets["NodeIDs"][n.NodeID] = true
			sets["CandidateNodeIDs"][n.NodeID] = true
		}
	}
	v.Nodes = nodes
	edges := []models.Edge{}
	for _, e := range v.Edges {
		if !sets["NodeIDs"][e.Source] || !sets["NodeIDs"][e.Target] {
			continue
		}
		hit := false
		for _, id := range e.EvidenceIDs {
			if sets["EvidenceIDs"][id] {
				hit = true
			}
		}
		if hit {
			edges = append(edges, e)
			sets["EdgeIDs"][e.EdgeID] = true
		}
	}
	v.Edges = edges
	for _, e := range v.Events {
		sets["EventIDs"][e.EventID] = true
	}
	for _, d := range v.Deals {
		sets["DealIDs"][d.DealID] = true
	}
	pruneProjection(reflect.ValueOf(&v).Elem(), s.AsOf, sets)
	clearUnprovedParticipants(reflect.ValueOf(&v).Elem())
	return v, nil
}
func clearUnprovedParticipants(v reflect.Value) {
	if v.Kind() == reflect.Pointer {
		if !v.IsNil() {
			clearUnprovedParticipants(v.Elem())
		}
		return
	}
	if v.Kind() == reflect.Struct {
		if v.Type() == reflect.TypeOf(models.ParticipantRef{}) && v.FieldByName("EvidenceIDs").Len() == 0 {
			v.FieldByName("NodeID").SetZero()
			v.FieldByName("VerificationState").SetString("unresolved")
		}
		for i := 0; i < v.NumField(); i++ {
			clearUnprovedParticipants(v.Field(i))
		}
	}
	if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			clearUnprovedParticipants(v.Index(i))
		}
	}
}
func fullPrimaryDeal(raw fullSelection, v visibleContext, id string, s models.SnapshotContext) (models.DealFacts, error) {
	account, exists := raw.Manifest.DealAccountIDs[id]
	if !exists {
		return models.DealFacts{}, &RepositoryError{Code: NotFound}
	}
	if !s.Access.AllowsDeal(id, account) {
		return models.DealFacts{}, &RepositoryError{Code: AccessDenied}
	}
	at, ok := raw.Manifest.DealCreatedAt[id]
	if !ok {
		return models.DealFacts{}, &RepositoryError{Code: DataNotReady}
	}
	if at > s.AsOf {
		return models.DealFacts{}, &RepositoryError{Code: NotVisible}
	}
	for _, d := range v.Deals {
		if d.DealID == id && d.AccountID == account {
			return d, nil
		}
	}
	return models.DealFacts{}, &RepositoryError{Code: DataNotReady}
}

// Restrict account context to the primary account and native deal events;
// authorizing several deals must not mix their timelines into one response.
func primaryFullView(v visibleContext, account, id string) visibleContext {
	events := []models.Event{}
	for _, e := range v.Events {
		hit := false
		for _, a := range e.AccountIDs {
			if a == account {
				hit = true
			}
		}
		if !hit {
			continue
		}
		if len(e.DealIDs) > 0 {
			hit = false
			for _, d := range e.DealIDs {
				if d == id {
					hit = true
				}
			}
			if !hit {
				continue
			}
		}
		events = append(events, e)
	}
	v.Events = events
	return v
}
