package models

type ActionTemplate struct {
	ActionID        string `json:"action_id"`
	ActionType      string `json:"action_type"`
	Definition      string `json:"definition"`
	TaxonomyVersion string `json:"taxonomy_version"`
}

// PolicyFacts belongs to one request/occurrence. It is not an authorization,
// suitability score, or consent inherited from an analog precedent.
type PolicyFacts struct {
	RequestedDiscountBPS Fact[int32]    `json:"requested_discount_bps"`
	RequestOccurrenceID  *string        `json:"request_occurrence_id"`
	DecisionID           *string        `json:"decision_id"`
	AppliedContractID    *string        `json:"applied_contract_id"`
	ApprovalStatus       Fact[string]   `json:"approval_status"`
	Approver             ParticipantRef `json:"approver"`
	ApprovedDiscountBPS  Fact[int32]    `json:"approved_discount_bps"`
	ConsentStatus        Fact[string]   `json:"consent_status"`
	ConsentTargetNodeID  *string        `json:"consent_target_node_id"`
	ConsentScope         Fact[string]   `json:"consent_scope"`
}
type ActionOccurrence struct {
	OccurrenceID    string           `json:"occurrence_id"`
	ActionID        string           `json:"action_id"`
	EventID         string           `json:"event_id"`
	Actor           ParticipantRef   `json:"actor"`
	Targets         []ParticipantRef `json:"targets"`
	EventAt         Date             `json:"event_at"`
	Status          Fact[string]     `json:"status"`
	RawStatus       *string          `json:"raw_status"`
	OutcomeObserved Fact[string]     `json:"outcome_observed"`
	OutcomeAt       *Date            `json:"outcome_at"`
	AccountIDs      []string         `json:"account_ids"`
	DealIDs         []string         `json:"deal_ids"`
	EvidenceIDs     []string         `json:"evidence_ids"`
	NodeIDs         []string         `json:"node_ids"`
	EdgeIDs         []string         `json:"edge_ids"`
	Policy          PolicyFacts      `json:"policy"`
}
type ActionPrecedent struct {
	Relation   string           `json:"relation"`
	Occurrence ActionOccurrence `json:"occurrence"`
}
type RelevanceReason struct {
	Code                 string   `json:"code"`
	Detail               string   `json:"detail"`
	DealEvidenceIDs      []string `json:"deal_evidence_ids"`
	PrecedentEvidenceIDs []string `json:"precedent_evidence_ids"`
}
type ActionCandidate struct {
	Template         ActionTemplate    `json:"template"`
	Precedents       []ActionPrecedent `json:"precedents"`
	RelevanceReasons []RelevanceReason `json:"relevance_reasons"`
	Limitations      []string          `json:"limitations"`
}
type CandidateOptions struct {
	RelevanceRuleVersion string
	Limit                int
	Cursor               *string
}
type ActionCandidatesResult struct {
	Meta                 Meta              `json:"meta"`
	Bounds               BoundInfo         `json:"bounds"`
	RelevanceRuleVersion string            `json:"relevance_rule_version"`
	ReturnedCount        int               `json:"returned_count"`
	TotalValidCount      *int              `json:"total_valid_count"`
	Items                []ActionCandidate `json:"items"`
}
type SelectedActionsResult struct {
	Meta                 Meta              `json:"meta"`
	RelevanceRuleVersion string            `json:"relevance_rule_version"`
	SelectedActionIDs    []string          `json:"selected_action_ids"`
	Items                []ActionCandidate `json:"items"`
	Evidence             []Evidence        `json:"evidence"`
}

// ActionComparisonResult contains model assessments, not approvals or source facts.
type ActionComparisonResult struct {
	Meta             Meta                   `json:"meta"`
	DealID           string                 `json:"deal_id"`
	AsOf             Date                   `json:"as_of"`
	ComparisonType   string                 `json:"comparison_type"`
	AssessmentStatus string                 `json:"assessment_status"`
	RubricVersion    string                 `json:"rubric_version"`
	Items            []ActionComparisonItem `json:"items"`
	Warning          string                 `json:"warning"`
}
type ActionComparisonItem struct {
	ActionID            string   `json:"action_id"`
	SuitabilityScore100 *int     `json:"suitability_score_100"`
	JEVRawScore         *float64 `json:"jev_raw_score"`
	JEVConfidence       *float64 `json:"jev_confidence"`
	JEVChoicePreference *float64 `json:"jev_choice_preference"`
	Rank                *int     `json:"rank"`
	EvidenceIDs         []string `json:"evidence_ids"`
	PrecedentIDs        []string `json:"precedent_ids"`
	PolicyFlags         []string `json:"policy_flags"`
	Unknowns            []string `json:"unknowns"`
}
