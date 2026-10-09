package models

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Provider types for the TypeSafe System One evaluation endpoint (JEV).
//
// Shapes follow the official reference verified on 2026-10-10:
//   - https://docs.typesafe.ai/api
//   - https://docs.typesafe.ai/primitives
//
// These are provider DTOs only. They are intentionally independent from O1
// graph/fact/evidence domain types and from Relio HTTP DTOs. A JEV answer is
// a model judgment over the supplied state, never source truth, a closing
// probability, an approval, or an action decision.

const (
	JEVQuestionTypeNoul   = "noul"
	JEVQuestionTypeChoice = "choice"
	JEVQuestionTypeScore  = "score"

	// JEVModelLatest is the documented flagship alias. The resolved model
	// version is reported by the provider in JEVResponse.Model.
	JEVModelLatest = "jev-latest"

	// Documented provider limits.
	JEVChoiceMaxOptions = 255
	JEVScoreMinLevels   = 2
	JEVScoreMaxLevels   = 10
)

// JEVRequest is the request body of POST /v1/systemone.
type JEVRequest struct {
	// State is the content to evaluate: a string, object, or array.
	State any `json:"state"`
	// Model is required by the provider. Empty means "use the client default".
	Model string `json:"model"`
	// Questions is keyed by caller-chosen IDs; answers return under the same IDs.
	Questions map[string]JEVQuestion `json:"questions"`
}

// JEVQuestion is one typed question. Exactly one criteria field may be set and
// it must match Type. Instructions and criteria values may be a string,
// object, or array per the provider reference.
type JEVQuestion struct {
	Type         string
	Instructions any

	// NoulCriteria is optional for noul questions.
	NoulCriteria *JEVNoulCriteria
	// ChoiceCriteria maps option -> description (nil description allowed).
	ChoiceCriteria map[string]any
	// ScoreCriteria is the ordered list of level descriptions. Level i is
	// reported by the provider under the string key strconv.Itoa(i).
	ScoreCriteria []any
}

// JEVNoulCriteria optionally describes what yes (near 1) and no (near 0) mean.
type JEVNoulCriteria struct {
	True  any `json:"true,omitempty"`
	False any `json:"false,omitempty"`
}

type jevQuestionWire struct {
	Type         string `json:"type"`
	Instructions any    `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"`
}

// MarshalJSON emits the provider wire shape {type, instructions, criteria}.
func (q JEVQuestion) MarshalJSON() ([]byte, error) {
	wire := jevQuestionWire{Type: q.Type, Instructions: q.Instructions}
	switch q.Type {
	case JEVQuestionTypeNoul:
		if q.NoulCriteria != nil {
			wire.Criteria = q.NoulCriteria
		}
	case JEVQuestionTypeChoice:
		wire.Criteria = q.ChoiceCriteria
	case JEVQuestionTypeScore:
		wire.Criteria = q.ScoreCriteria
	default:
		return nil, fmt.Errorf("unsupported JEV question type %q", q.Type)
	}
	return json.Marshal(wire)
}

// JEVResponse is the documented success body.
type JEVResponse struct {
	// Model is the provider-reported model that performed the evaluation
	// (for example a resolved version behind the jev-latest alias).
	Model   string               `json:"model"`
	Answers map[string]JEVAnswer `json:"answers"`
	// Usage is documented as required; token fields are kept as pointers so
	// absent values are not reported as zero.
	Usage *JEVUsage `json:"usage"`
}

// JEVUsage is token usage exactly as reported by the provider.
type JEVUsage struct {
	InputTokens  *int64 `json:"input_tokens,omitempty"`
	OutputTokens *int64 `json:"output_tokens,omitempty"`
}

// JEVAnswer is a typed answer. Which fields are required depends on Type:
//   - noul:   Noul in [0,1]. No confidence is documented for noul.
//   - choice: Choice, Probabilities over every option, Confidence.
//   - score:  Score (probability-weighted, may fall between levels), Legend,
//     Probabilities over level indexes, Confidence.
//
// Confidence is the provider's distribution-derived certainty, returned only
// when the provider sends it. It is not a business confidence or a closing
// probability.
type JEVAnswer struct {
	Type          string             `json:"type"`
	Noul          *float64           `json:"noul,omitempty"`
	Choice        *string            `json:"choice,omitempty"`
	Score         *float64           `json:"score,omitempty"`
	Legend        map[string]string  `json:"legend,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Confidence    *float64           `json:"confidence,omitempty"`
}

// UnmarshalJSON prevents null map entries from silently becoming known zero
// probabilities or empty legend strings under encoding/json's default behavior.
func (a *JEVAnswer) UnmarshalJSON(data []byte) error {
	type answer JEVAnswer
	var decoded answer
	wire := struct {
		*answer
		Legend        map[string]*string  `json:"legend"`
		Probabilities map[string]*float64 `json:"probabilities"`
	}{answer: &decoded}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	if wire.Legend != nil {
		decoded.Legend = make(map[string]string, len(wire.Legend))
		for key, text := range wire.Legend {
			if text == nil {
				return errors.New("JEV legend value must not be null")
			}
			decoded.Legend[key] = *text
		}
	}
	if wire.Probabilities != nil {
		decoded.Probabilities = make(map[string]float64, len(wire.Probabilities))
		for key, probability := range wire.Probabilities {
			if probability == nil {
				return errors.New("JEV probability must not be null")
			}
			decoded.Probabilities[key] = *probability
		}
	}
	*a = JEVAnswer(decoded)
	return nil
}
