package models

// DealAssessment is a model judgment, never a source fact or closing probability.
type DealAssessment struct {
	Meta           Meta      `json:"meta"`
	Status         string    `json:"status"`
	RubricVersion  string    `json:"rubric_version"`
	Explanation    string    `json:"explanation"`
	Unknowns       []string  `json:"unknowns"`
	Readiness100   *int      `json:"readiness_100"`
	ReadinessLabel *string   `json:"readiness_label"`
	JEVRawScore    *float64  `json:"jev_raw_score"`
	JEVConfidence  *float64  `json:"jev_confidence"`
	EvidenceIDs    []string  `json:"evidence_ids"`
	ProviderModel  *string   `json:"provider_model"`
	Usage          *JEVUsage `json:"usage"`
}
