package models

// Date is a calendar token, not a synthesized source timestamp.
// Validate at repository boundaries with ParseDate.
type Date string

// AccessScope is supplied by authorized server code, never by a browser.
// Analog permissions do not grant primary deal/account access.
type AccessScope struct {
	AllowedAccountIDs       []string `json:"allowed_account_ids"`
	AllowedDealIDs          []string `json:"allowed_deal_ids"`
	AllowedAnalogAccountIDs []string `json:"allowed_analog_account_ids"`
	AllowCompanyEvidence    bool     `json:"allow_company_evidence"`
}
type SnapshotContext struct {
	AsOf            Date        `json:"as_of"`
	DatasetVersion  string      `json:"dataset_version"`
	ContractVersion string      `json:"contract_version"`
	CalendarZone    string      `json:"calendar_zone"`
	ContextID       string      `json:"context_id"`
	Access          AccessScope `json:"-"`
}
type Unknown struct {
	Path         string   `json:"path"`
	Reason       string   `json:"reason"`
	ReferenceIDs []string `json:"reference_ids"`
}
type Meta struct {
	ContractVersion string    `json:"contract_version"`
	DatasetVersion  string    `json:"dataset_version"`
	AsOf            Date      `json:"as_of"`
	MaxAsOf         Date      `json:"max_as_of"`
	CalendarZone    string    `json:"calendar_zone"`
	ContextID       string    `json:"context_id"`
	DataState       string    `json:"data_state"`
	Limitations     []string  `json:"limitations"`
	Unknowns        []Unknown `json:"unknowns"`
}
type BoundInfo struct {
	Truncated         bool     `json:"truncated"`
	TruncationReasons []string `json:"truncation_reasons"`
	NextCursor        *string  `json:"next_cursor"`
}
