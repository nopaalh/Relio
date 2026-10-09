package repository

import (
	"context"
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"sort"
	"strings"
)

func (r *Neo4jFullContextRepository) FindFacts(ctx context.Context, id string, s models.SnapshotContext) (models.DealFactsResult, error) {
	raw, v, err := r.load(ctx, s)
	if err != nil {
		return models.DealFactsResult{}, err
	}
	d, err := fullPrimaryDeal(raw, v, id, s)
	if err != nil {
		return models.DealFactsResult{}, err
	}
	return models.DealFactsResult{Meta: v.Meta, Deal: d}, nil
}
func (r *Neo4jFullContextRepository) ListFacts(ctx context.Context, s models.SnapshotContext, o models.DealListOptions) (models.DealFactsPage, error) {
	limit, err := NormalizePageLimit(o.Limit)
	if err != nil {
		return models.DealFactsPage{}, err
	}
	accounts, err := sortedFilter(o.AccountIDs)
	if err != nil {
		return models.DealFactsPage{}, err
	}
	types, err := sortedFilter(o.DealTypes)
	if err != nil {
		return models.DealFactsPage{}, err
	}
	key := filterKey("deals", accounts, types, limit)
	after, err := cursorAfter(o.Cursor, s.ContextID, key)
	if err != nil {
		return models.DealFactsPage{}, err
	}
	_, v, err := r.load(ctx, s)
	if err != nil {
		return models.DealFactsPage{}, err
	}
	out := models.DealFactsPage{Meta: v.Meta, Bounds: emptyBounds(), Items: []models.DealFacts{}}
	sort.Slice(v.Deals, func(i, j int) bool { return v.Deals[i].DealID < v.Deals[j].DealID })
	validAfter := after == ""
	for _, d := range v.Deals {
		if !matches(accounts, d.AccountID) {
			continue
		}
		if len(types) > 0 && (d.DealType.Value == nil || !matches(types, *d.DealType.Value)) {
			continue
		}
		if d.DealID == after {
			validAfter = true
		}
		if d.DealID > after {
			out.Items = append(out.Items, d)
		}
	}
	if !validAfter {
		return models.DealFactsPage{}, &RepositoryError{Code: InvalidQuery}
	}
	if len(out.Items) > limit {
		out.Items = out.Items[:limit]
		out.Bounds.Truncated = true
		out.Bounds.TruncationReasons = []string{"page_limit"}
		out.Bounds.NextCursor = nextCursor(s.ContextID, key, out.Items[len(out.Items)-1].DealID)
	}
	return out, nil
}
func (r *Neo4jFullContextRepository) ReadGraph(ctx context.Context, id string, s models.SnapshotContext, o models.GraphOptions) (models.GraphResult, error) {
	o, err := NormalizeGraphOptions(o)
	if err != nil {
		return models.GraphResult{}, err
	}
	q := fullQueryFor(s, "context")
	if o.FocusEventID != nil {
		q.LookupID = *o.FocusEventID
		q.LookupKind = "events"
	}
	raw, v, err := r.loadFull(ctx, s, q)
	if err != nil {
		return models.GraphResult{}, err
	}
	if _, err = fullPrimaryDeal(raw, v, id, s); err != nil {
		return models.GraphResult{}, err
	}
	v, err = fullViewForDeal(raw, s, id)
	if err != nil {
		return models.GraphResult{}, err
	}
	roots := []string{"account:" + raw.Manifest.DealAccountIDs[id], "deal:" + id}
	if o.FocusEventID != nil {
		found := false
		for _, e := range v.Events {
			if e.EventID == *o.FocusEventID {
				found = true
				roots = append([]string{e.EventID}, e.NodeIDs...)
				roots = append(roots, "account:"+raw.Manifest.DealAccountIDs[id], "deal:"+id)
				break
			}
		}
		if !found {
			for _, record := range raw.Records {
				var e models.Event
				if record.Kind != "events" {
					continue
				}
				_ = json.Unmarshal(record.Payload, &e)
				if e.EventID == *o.FocusEventID {
					if !fullRecordAllowed(record, s, false) {
						return models.GraphResult{}, &RepositoryError{Code: AccessDenied}
					}
					return models.GraphResult{}, &RepositoryError{Code: NotVisible}
				}
			}
			return models.GraphResult{}, &RepositoryError{Code: NotFound}
		}
	}
	distance := map[string]int{}
	queue := []string{}
	for _, root := range roots {
		distance[root] = 0
		queue = append(queue, root)
	}
	for len(queue) > 0 {
		root := queue[0]
		queue = queue[1:]
		if distance[root] >= o.Depth {
			continue
		}
		for _, e := range v.Edges {
			other := ""
			if e.Source == root {
				other = e.Target
			} else if e.Target == root {
				other = e.Source
			}
			if other != "" {
				if _, ok := distance[other]; !ok {
					distance[other] = distance[root] + 1
					queue = append(queue, other)
				}
			}
		}
	}
	sort.Slice(v.Nodes, func(i, j int) bool {
		if o.FocusEventID != nil && (v.Nodes[i].NodeID == *o.FocusEventID || v.Nodes[j].NodeID == *o.FocusEventID) {
			return v.Nodes[i].NodeID == *o.FocusEventID
		}
		di, oki := distance[v.Nodes[i].NodeID]
		dj, okj := distance[v.Nodes[j].NodeID]
		if oki != okj {
			return oki
		}
		if di != dj {
			return di < dj
		}
		return v.Nodes[i].NodeID < v.Nodes[j].NodeID
	})
	sort.Slice(v.Edges, func(i, j int) bool { return v.Edges[i].EdgeID < v.Edges[j].EdgeID })
	out := models.GraphResult{Meta: v.Meta, Bounds: emptyBounds(), Nodes: []models.Node{}, Edges: []models.Edge{}}
	ns := map[string]bool{}
	truncated := false
	for _, n := range v.Nodes {
		if _, ok := distance[n.NodeID]; !ok {
			continue
		}
		if len(out.Nodes) >= o.MaxNodes {
			truncated = true
			continue
		}
		out.Nodes = append(out.Nodes, n)
		ns[n.NodeID] = true
	}
	for _, e := range v.Edges {
		_, sourceCandidate := distance[e.Source]
		_, targetCandidate := distance[e.Target]
		if !sourceCandidate || !targetCandidate {
			continue
		}
		if !ns[e.Source] || !ns[e.Target] || len(out.Edges) >= o.MaxEdges {
			truncated = true
			continue
		}
		out.Edges = append(out.Edges, e)
	}
	if truncated {
		out.Bounds.Truncated = true
		out.Bounds.TruncationReasons = []string{"graph_limits"}
	}
	return out, nil
}
func (r *Neo4jFullContextRepository) ReadTimeline(ctx context.Context, id string, s models.SnapshotContext, o models.TimelineOptions) (models.TimelineResult, error) {
	limit, err := NormalizePageLimit(o.Limit)
	if err != nil {
		return models.TimelineResult{}, err
	}
	filters := [][]string{o.EventIDs, o.EventTypes, o.ActorNodeIDs, o.Statuses}
	for i := range filters {
		filters[i], err = sortedFilter(filters[i])
		if err != nil {
			return models.TimelineResult{}, err
		}
	}
	key := filterKey("timeline", id, filters, limit)
	after, err := cursorAfter(o.Cursor, s.ContextID, key)
	if err != nil {
		return models.TimelineResult{}, err
	}
	raw, v, err := r.load(ctx, s)
	if err != nil {
		return models.TimelineResult{}, err
	}
	if _, err = fullPrimaryDeal(raw, v, id, s); err != nil {
		return models.TimelineResult{}, err
	}
	v, err = fullViewForDeal(raw, s, id)
	if err != nil {
		return models.TimelineResult{}, err
	}
	sort.Slice(v.Events, func(i, j int) bool { return eventOrder(v.Events[i]) < eventOrder(v.Events[j]) })
	out := models.TimelineResult{Meta: v.Meta, Bounds: emptyBounds(), Events: []models.Event{}}
	validAfter := after == ""
	for _, e := range v.Events {
		if !matches(filters[0], e.EventID) || !matches(filters[1], e.EventType) {
			continue
		}
		if len(filters[3]) > 0 && (e.Status.Value == nil || !matches(filters[3], *e.Status.Value)) {
			continue
		}
		if len(filters[2]) > 0 {
			hit := false
			for _, a := range e.Actors {
				if a.NodeID != nil && matches(filters[2], *a.NodeID) {
					hit = true
				}
			}
			if !hit {
				continue
			}
		}
		order := eventOrder(e)
		if order == after {
			validAfter = true
		}
		if order > after {
			out.Events = append(out.Events, e)
		}
	}
	if !validAfter {
		return models.TimelineResult{}, &RepositoryError{Code: InvalidQuery}
	}
	if len(out.Events) > limit {
		out.Events = out.Events[:limit]
		out.Bounds.Truncated = true
		out.Bounds.TruncationReasons = []string{"page_limit"}
		out.Bounds.NextCursor = nextCursor(s.ContextID, key, eventOrder(out.Events[len(out.Events)-1]))
	}
	return out, nil
}
func fullViewForDeal(raw fullSelection, s models.SnapshotContext, id string) (visibleContext, error) {
	s.Access.AllowedAccountIDs = []string{raw.Manifest.DealAccountIDs[id]}
	s.Access.AllowedDealIDs = []string{id}
	s.Access.AllowedAnalogAccountIDs = []string{}
	v, err := projectFull(raw, s)
	if err != nil {
		return v, err
	}
	return primaryFullView(v, raw.Manifest.DealAccountIDs[id], id), nil
}
func (r *Neo4jFullContextRepository) ReadEvidence(ctx context.Context, id string, s models.SnapshotContext) (models.EvidenceResult, error) {
	if id == "" {
		return models.EvidenceResult{}, &RepositoryError{Code: InvalidQuery}
	}
	q := fullQueryFor(s, "evidence")
	q.LookupID = id
	raw, v, err := r.loadFull(ctx, s, q)
	if err != nil {
		return models.EvidenceResult{}, err
	}
	if strings.HasPrefix(id, "ev:usage:") {
		return fullUsageEvidence(raw, id, s)
	}
	for _, e := range v.Evidence {
		if e.EvidenceID == id {
			return models.EvidenceResult{Meta: v.Meta, Evidence: e}, nil
		}
	}
	for _, r := range raw.Records {
		if r.Kind == "evidence" && r.ID == id {
			if !fullRecordAllowed(r, s, false) {
				return models.EvidenceResult{}, &RepositoryError{Code: AccessDenied}
			}
			return models.EvidenceResult{}, &RepositoryError{Code: NotVisible}
		}
	}
	return models.EvidenceResult{}, &RepositoryError{Code: NotFound}
}

var _ DealFactsRepository = (*Neo4jFullContextRepository)(nil)
var _ GraphRepository = (*Neo4jFullContextRepository)(nil)
var _ TimelineRepository = (*Neo4jFullContextRepository)(nil)
var _ EvidenceRepository = (*Neo4jFullContextRepository)(nil)
