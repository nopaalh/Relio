package repository

import (
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"reflect"
)

type visibleContext struct {
	Meta     models.Meta
	Deals    []models.DealFacts
	Nodes    []models.Node
	Edges    []models.Edge
	Events   []models.Event
	Evidence []models.Evidence
}

func projectContext(raw storedContext, s models.SnapshotContext) (visibleContext, error) {
	var out visibleContext
	if err := ValidateSnapshot(s); err != nil {
		return out, err
	}
	if !raw.Manifest.Ready || raw.Manifest.DatasetVersion != s.DatasetVersion || raw.Manifest.SchemaVersion != neo4jSchemaVersion {
		return out, &RepositoryError{Code: DataNotReady}
	}
	if err := validateStoredContext(raw); err != nil {
		return out, &RepositoryError{Code: DataNotReady, Cause: err}
	}
	if !s.Access.AllowsAccount("P04") {
		return out, &RepositoryError{Code: AccessDenied}
	}
	// Deep copy: never mutate shared records while projecting concurrent dates.
	bytes, err := json.Marshal(raw)
	if err != nil {
		return out, &RepositoryError{Code: DataNotReady, Cause: err}
	}
	var copy storedContext
	if err = json.Unmarshal(bytes, &copy); err != nil {
		return out, &RepositoryError{Code: DataNotReady, Cause: err}
	}
	out = visibleContext{Meta: models.Meta{ContractVersion: s.ContractVersion, DatasetVersion: s.DatasetVersion, AsOf: s.AsOf, MaxAsOf: MaxAsOf, CalendarZone: s.CalendarZone, ContextID: s.ContextID, DataState: "partial", Limitations: []string{"coverage:DL-004/P04 only", "historical_email_identity_unresolved", "employment_enrichment_not_loaded"}, Unknowns: []models.Unknown{}}, Deals: []models.DealFacts{}, Nodes: []models.Node{}, Edges: []models.Edge{}, Events: []models.Event{}, Evidence: []models.Evidence{}}
	for _, d := range copy.Deals {
		if d.AvailableFrom <= s.AsOf && s.Access.AllowsDeal(d.Value.DealID, d.Value.AccountID) {
			out.Deals = append(out.Deals, d.Value)
		}
	}
	for _, e := range copy.Evidence {
		v := e.Value
		allowed := v.ScopeKind == "account" && len(v.AccountIDs) == 1 && s.Access.AllowsAccount(v.AccountIDs[0])
		if v.ScopeKind == "deal" && len(v.AccountIDs) == 1 && len(v.DealIDs) == 1 {
			allowed = s.Access.AllowsDeal(v.DealIDs[0], v.AccountIDs[0])
		}
		if allowed && e.AvailableFrom <= s.AsOf {
			out.Evidence = append(out.Evidence, v)
		}
	}
	for _, e := range copy.Events {
		if e.EventAt <= s.AsOf && e.ScopeKind == "account" && len(e.AccountIDs) == 1 && s.Access.AllowsAccount(e.AccountIDs[0]) {
			out.Events = append(out.Events, e)
		}
	}
	for _, n := range copy.Nodes {
		v := n.Value
		if n.AvailableFrom <= s.AsOf && validAt(v.Validity, s.AsOf) && (v.NodeType != "deal" || s.Access.AllowsDeal(v.EntityID, "P04")) {
			maskValidity(&v.Validity, s.AsOf)
			out.Nodes = append(out.Nodes, v)
		}
	}
	ns := map[string]bool{}
	for _, n := range out.Nodes {
		ns[n.NodeID] = true
	}
	for _, e := range copy.Edges {
		v := e.Value
		if e.AvailableFrom <= s.AsOf && validAt(v.Validity, s.AsOf) && ns[v.Source] && ns[v.Target] {
			maskValidity(&v.Validity, s.AsOf)
			out.Edges = append(out.Edges, v)
		}
	}
	sets := map[string]map[string]bool{"EvidenceIDs": {}, "RecordEvidenceIDs": {}, "NodeIDs": ns, "CandidateNodeIDs": ns, "EdgeIDs": {}, "EventIDs": {}, "DealIDs": {}, "AccountIDs": {"P04": true}}
	for _, e := range out.Evidence {
		sets["EvidenceIDs"][e.EvidenceID] = true
		sets["RecordEvidenceIDs"][e.EvidenceID] = true
	}
	for _, e := range out.Edges {
		sets["EdgeIDs"][e.EdgeID] = true
	}
	for _, e := range out.Events {
		sets["EventIDs"][e.EventID] = true
	}
	for _, d := range out.Deals {
		sets["DealIDs"][d.DealID] = true
	}
	pruneProjection(reflect.ValueOf(&out).Elem(), s.AsOf, sets)
	for _, d := range out.Deals {
		for _, item := range []struct {
			path  string
			known bool
		}{{"stage", d.Stage.Value != nil}, {"owner_id", d.OwnerID.Value != nil}, {"potential_acv_idr", d.PotentialACVIDR.Value != nil}, {"planned_outlets", d.PlannedOutlets.Value != nil}} {
			if !item.known {
				out.Meta.Unknowns = append(out.Meta.Unknowns, models.Unknown{Path: "deal." + item.path, Reason: "snapshot_only_no_history", ReferenceIDs: []string{}})
			}
		}
	}
	return out, nil
}
func validAt(v models.Validity, asOf models.Date) bool {
	return (v.ValidFrom == nil || *v.ValidFrom <= asOf) && (v.ValidTo == nil || asOf < *v.ValidTo)
}
func maskValidity(v *models.Validity, asOf models.Date) {
	if v.ValidTo != nil && *v.ValidTo > asOf {
		v.ValidTo = nil
		reason := "end_unknown_at_snapshot"
		v.OpenEndReason = &reason
	}
}
func pruneProjection(v reflect.Value, asOf models.Date, sets map[string]map[string]bool) {
	if v.Kind() == reflect.Pointer {
		if !v.IsNil() {
			pruneProjection(v.Elem(), asOf, sets)
		}
		return
	}
	switch v.Kind() {
	case reflect.Struct:
		// Every Fact has these fields; mask snapshot facts before pruning refs.
		basis := v.FieldByName("TemporalBasis")
		state := v.FieldByName("State")
		value := v.FieldByName("Value")
		if basis.IsValid() && state.IsValid() && value.IsValid() && basis.Kind() == reflect.String && value.Kind() == reflect.Pointer && basis.String() == "snapshot" && asOf < MaxAsOf {
			value.SetZero()
			state.SetString("snapshot_only")
			v.FieldByName("EvidenceIDs").Set(reflect.MakeSlice(v.FieldByName("EvidenceIDs").Type(), 0, 0))
			v.FieldByName("Limitations").Set(reflect.ValueOf([]string{"current_snapshot_value_has_no_historical_support"}))
		}
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			name := v.Type().Field(i).Name
			if set, ok := sets[name]; ok && f.Kind() == reflect.Slice && f.Type().Elem().Kind() == reflect.String {
				kept := reflect.MakeSlice(f.Type(), 0, f.Len())
				for j := 0; j < f.Len(); j++ {
					if set[f.Index(j).String()] {
						kept = reflect.Append(kept, f.Index(j))
					}
				}
				f.Set(kept)
			}
			pruneProjection(f, asOf, sets)
		}
		if v.Type() == reflect.TypeOf(models.ParticipantRef{}) {
			chosen := v.FieldByName("NodeID")
			if !chosen.IsNil() && !sets["NodeIDs"][chosen.Elem().String()] {
				chosen.SetZero()
				v.FieldByName("VerificationState").SetString("unresolved")
			}
		}
		if state.IsValid() && value.IsValid() && value.Kind() == reflect.Pointer && state.String() == "known" && v.FieldByName("EvidenceIDs").Len() == 0 {
			value.SetZero()
			state.SetString("unknown")
			v.FieldByName("Limitations").Set(reflect.ValueOf([]string{"proof_not_visible_in_context"}))
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			pruneProjection(v.Index(i), asOf, sets)
		}
	}
}
