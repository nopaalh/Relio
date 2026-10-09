package models

// DealAssessment is a model judgment, never a source fact or closing probability.
type DealAssessment struct {
	Meta                Meta                `json:"meta"`
	Status              string              `json:"status"`
	RubricVersion       string              `json:"rubric_version"`
	Explanation         string              `json:"explanation"`
	Unknowns            []string            `json:"unknowns"`
	Readiness100        *int                `json:"readiness_100"`
	ReadinessLabel      *string             `json:"readiness_label"`
	JEVRawScore         *float64            `json:"jev_raw_score"`
	JEVConfidence       *float64            `json:"jev_confidence"`
	EvidenceIDs         []string            `json:"evidence_ids"`
	ProviderModel       *string             `json:"provider_model"`
	Usage               *JEVUsage           `json:"usage"`
	Attractiveness100   *int                `json:"attractiveness_100"`
	ACVRelative100      *int                `json:"acv_relative_100"`
	ACVDenominatorIDR   *int64              `json:"acv_denominator_idr"`
	StrategicOutlets100 *int                `json:"strategic_outlets_100"`
	Urgency100          *int                `json:"urgency_100"`
	Blocker             *string             `json:"blocker"`
	BlockerEvidenceIDs  []string            `json:"blocker_evidence_ids"`
	Coverage            *int                `json:"coverage"`
	CoverageDimensions  []CoverageDimension `json:"coverage_dimensions"`
}

type CoverageDimension struct {
	Key         string   `json:"key"`
	Present     bool     `json:"present"`
	EvidenceIDs []string `json:"evidence_ids"`
}
