package models

// Normalize functions copy collections for JSON without changing fact values or states.
func httpCollection[T any](items []T) []T { return append([]T{}, items...) }

func httpFact[T any](fact Fact[T]) Fact[T] {
	fact.EvidenceIDs = httpCollection(fact.EvidenceIDs)
	fact.Limitations = httpCollection(fact.Limitations)
	return fact
}

func httpMeta(meta Meta) Meta {
	meta.Limitations = httpCollection(meta.Limitations)
	meta.Unknowns = httpCollection(meta.Unknowns)
	for i := range meta.Unknowns {
		meta.Unknowns[i].ReferenceIDs = httpCollection(meta.Unknowns[i].ReferenceIDs)
	}
	return meta
}

func httpBounds(bounds BoundInfo) BoundInfo {
	bounds.TruncationReasons = httpCollection(bounds.TruncationReasons)
	return bounds
}

func httpDeal(deal DealFacts) DealFacts {
	deal.DealType = httpFact(deal.DealType)
	deal.OwnerID = httpFact(deal.OwnerID)
	deal.Stage = httpFact(deal.Stage)
	deal.StageSince = httpFact(deal.StageSince)
	deal.CreatedAt = httpFact(deal.CreatedAt)
	deal.PlannedOutlets = httpFact(deal.PlannedOutlets)
	deal.PotentialACVIDR = httpFact(deal.PotentialACVIDR)
	deal.Status = httpFact(deal.Status)
	deal.RecordEvidenceIDs = httpCollection(deal.RecordEvidenceIDs)
	return deal
}

func httpParticipant(participant ParticipantRef) ParticipantRef {
	participant.CandidateNodeIDs = httpCollection(participant.CandidateNodeIDs)
	participant.EvidenceIDs = httpCollection(participant.EvidenceIDs)
	return participant
}

func httpParticipants(participants []ParticipantRef) []ParticipantRef {
	participants = httpCollection(participants)
	for i := range participants {
		participants[i] = httpParticipant(participants[i])
	}
	return participants
}

func NormalizeDealPage(result DealFactsPage) DealFactsPage {
	result.Meta, result.Bounds = httpMeta(result.Meta), httpBounds(result.Bounds)
	result.Items = httpCollection(result.Items)
	for i := range result.Items { result.Items[i] = httpDeal(result.Items[i]) }
	return result
}

func NormalizeDealResult(result DealFactsResult) DealFactsResult {
	result.Meta, result.Deal = httpMeta(result.Meta), httpDeal(result.Deal)
	return result
}

func NormalizeGraphResult(result GraphResult) GraphResult {
	result.Meta, result.Bounds = httpMeta(result.Meta), httpBounds(result.Bounds)
	result.Nodes, result.Edges = httpCollection(result.Nodes), httpCollection(result.Edges)
	for i := range result.Nodes {
		node := &result.Nodes[i]
		node.Label = httpFact(node.Label)
		node.EvidenceIDs, node.EventIDs = httpCollection(node.EvidenceIDs), httpCollection(node.EventIDs)
		if node.Person != nil {
			person := httpParticipant(*node.Person)
			node.Person = &person
		}
	}
	for i := range result.Edges {
		result.Edges[i].EvidenceIDs = httpCollection(result.Edges[i].EvidenceIDs)
		result.Edges[i].EventIDs = httpCollection(result.Edges[i].EventIDs)
	}
	return result
}

func NormalizeTimelineResult(result TimelineResult) TimelineResult {
	result.Meta, result.Bounds = httpMeta(result.Meta), httpBounds(result.Bounds)
	result.Events = httpCollection(result.Events)
	for i := range result.Events {
		event := &result.Events[i]
		event.Status, event.Summary = httpFact(event.Status), httpFact(event.Summary)
		event.Actors, event.Targets, event.Participants = httpParticipants(event.Actors), httpParticipants(event.Targets), httpParticipants(event.Participants)
		event.DealIDs, event.AccountIDs = httpCollection(event.DealIDs), httpCollection(event.AccountIDs)
		event.NodeIDs, event.EdgeIDs, event.EvidenceIDs = httpCollection(event.NodeIDs), httpCollection(event.EdgeIDs), httpCollection(event.EvidenceIDs)
	}
	return result
}

func NormalizeEvidenceResult(result EvidenceResult) EvidenceResult {
	result.Meta = httpMeta(result.Meta)
	evidence := &result.Evidence
	keys := make(map[string]string, len(evidence.SourceKey))
	for key, value := range evidence.SourceKey { keys[key] = value }
	evidence.SourceKey = keys
	evidence.DealIDs, evidence.AccountIDs = httpCollection(evidence.DealIDs), httpCollection(evidence.AccountIDs)
	evidence.EventIDs, evidence.EdgeIDs, evidence.OccurrenceIDs = httpCollection(evidence.EventIDs), httpCollection(evidence.EdgeIDs), httpCollection(evidence.OccurrenceIDs)
	return result
}
