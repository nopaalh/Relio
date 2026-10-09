package models

// DealFacts is additive: the legacy Deal model remains unchanged.
type DealFacts struct {
	DealID            string       `json:"deal_id"`
	AccountID         string       `json:"account_id"`
	DealType          Fact[string] `json:"deal_type"`
	OwnerID           Fact[string] `json:"owner_id"`
	Stage             Fact[string] `json:"stage"`
	StageSince        Fact[Date]   `json:"stage_since"`
	CreatedAt         Fact[Date]   `json:"created_at"`
	PlannedOutlets    Fact[int]    `json:"planned_outlets"`
	PotentialACVIDR   Fact[int64]  `json:"potential_acv_idr"`
	Status            Fact[string] `json:"status"`
	RecordEvidenceIDs []string     `json:"record_evidence_ids"`
}
type DealListOptions struct {
	AccountIDs []string
	DealTypes  []string
	Limit      int
	Cursor     *string
}
type DealFactsPage struct {
	Meta   Meta        `json:"meta"`
	Bounds BoundInfo   `json:"bounds"`
	Items  []DealFacts `json:"items"`
}
type DealFactsResult struct {
	Meta Meta      `json:"meta"`
	Deal DealFacts `json:"deal"`
}
