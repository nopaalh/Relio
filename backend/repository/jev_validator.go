package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/nopaalh/Relio/backend/models"
)

// JEVProbabilityTolerance is the absolute tolerance used when checking that a
// provider probability distribution sums to 1 and that a Score equals the
// probability-weighted level index.
//
// The provider documents probabilities as "floats that sum to 1" and its
// examples are serialized with two decimals. 0.02 absorbs float/serialization
// rounding for small option sets while still rejecting distributions that are
// clearly not normalized (for example 0.9 or 1.1). It is a schema guard, not a
// statement about model calibration.
const JEVProbabilityTolerance = 0.02

// jevArgmaxEpsilon allows ties when checking that `choice` is the
// highest-probability option.
const jevArgmaxEpsilon = 1e-9

// ValidateJEVRequest checks a request locally before any paid call is made.
// Non-empty string state/instructions and the 2-option Choice minimum are
// Relio policies; the other shapes and bounds are documented provider limits.
// Detailed validator errors may include caller/provider IDs: do not expose them.
func ValidateJEVRequest(req models.JEVRequest) error {
	if err := checkJEVContent(req.State, false, false); err != nil {
		return fmt.Errorf("state: %w", err)
	}
	if strings.TrimSpace(req.Model) == "" {
		return fmt.Errorf("model is required")
	}
	if len(req.Questions) == 0 {
		return fmt.Errorf("at least one question is required")
	}
	for _, id := range sortedKeys(req.Questions) {
		q := req.Questions[id]
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("question id must not be empty")
		}
		if err := checkJEVContent(q.Instructions, false, false); err != nil {
			return fmt.Errorf("question %q instructions: %w", id, err)
		}
		switch q.Type {
		case models.JEVQuestionTypeNoul:
			if q.ChoiceCriteria != nil || q.ScoreCriteria != nil {
				return fmt.Errorf("question %q: noul accepts only noul criteria", id)
			}
			if q.NoulCriteria != nil {
				for _, description := range []any{q.NoulCriteria.True, q.NoulCriteria.False} {
					if description == nil { // Unset optional descriptions are omitted on the wire.
						continue
					}
					if err := checkJEVContent(description, false, true); err != nil {
						return fmt.Errorf("question %q noul criteria: %w", id, err)
					}
				}
			}
		case models.JEVQuestionTypeChoice:
			if q.NoulCriteria != nil || q.ScoreCriteria != nil {
				return fmt.Errorf("question %q: choice accepts only choice criteria", id)
			}
			n := len(q.ChoiceCriteria)
			if n < 2 || n > models.JEVChoiceMaxOptions {
				return fmt.Errorf("question %q: choice needs 2..%d options, got %d", id, models.JEVChoiceMaxOptions, n)
			}
			for _, opt := range sortedKeys(q.ChoiceCriteria) {
				if strings.TrimSpace(opt) == "" {
					return fmt.Errorf("question %q: choice option must not be empty", id)
				}
				if err := checkJEVContent(q.ChoiceCriteria[opt], true, true); err != nil {
					return fmt.Errorf("question %q choice description: %w", id, err)
				}
			}
		case models.JEVQuestionTypeScore:
			if q.NoulCriteria != nil || q.ChoiceCriteria != nil {
				return fmt.Errorf("question %q: score accepts only score criteria", id)
			}
			n := len(q.ScoreCriteria)
			if n < models.JEVScoreMinLevels || n > models.JEVScoreMaxLevels {
				return fmt.Errorf("question %q: score needs %d..%d levels, got %d", id, models.JEVScoreMinLevels, models.JEVScoreMaxLevels, n)
			}
			for i, level := range q.ScoreCriteria {
				if err := checkJEVContent(level, false, true); err != nil {
					return fmt.Errorf("question %q score level %d: %w", id, i, err)
				}
			}
		default:
			return fmt.Errorf("question %q: unsupported type %q", id, q.Type)
		}
	}
	return nil
}

// Check the wire kind, not Go's dynamic type: typed nils and RawMessage can
// encode null/scalars, while structs and pointers can encode valid objects.
func checkJEVContent(value any, allowNull, allowEmptyString bool) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("must be JSON-encodable")
	}
	raw = bytes.TrimSpace(raw)
	switch raw[0] {
	case '{', '[':
		return nil
	case '"':
		if !allowEmptyString {
			var text string
			if err := json.Unmarshal(raw, &text); err != nil {
				return fmt.Errorf("invalid JSON string")
			}
			if strings.TrimSpace(text) == "" {
				return fmt.Errorf("must not be empty")
			}
		}
		return nil
	case 'n':
		if allowNull {
			return nil
		}
	}
	return fmt.Errorf("must be a string, object, or array")
}

// ValidateJEVResponse checks a decoded provider response against the questions
// that were sent. Passing validation means the response is schema-consistent
// with the documented contract; it does not mean the judgment is correct or
// supported by evidence.
func ValidateJEVResponse(questions map[string]models.JEVQuestion, resp models.JEVResponse) error {
	if strings.TrimSpace(resp.Model) == "" {
		return fmt.Errorf("model is missing")
	}
	if resp.Answers == nil {
		return fmt.Errorf("answers are missing")
	}
	if resp.Usage == nil {
		return fmt.Errorf("usage is missing")
	}
	if t := resp.Usage.InputTokens; t != nil && *t < 0 {
		return fmt.Errorf("usage.input_tokens is negative")
	}
	if t := resp.Usage.OutputTokens; t != nil && *t < 0 {
		return fmt.Errorf("usage.output_tokens is negative")
	}

	for _, id := range sortedKeys(resp.Answers) {
		if _, ok := questions[id]; !ok {
			return fmt.Errorf("answer %q does not match any question", id)
		}
	}
	for _, id := range sortedKeys(questions) {
		answer, ok := resp.Answers[id]
		if !ok {
			return fmt.Errorf("answer for question %q is missing", id)
		}
		if err := validateJEVAnswer(questions[id], answer); err != nil {
			return fmt.Errorf("answer %q: %w", id, err)
		}
	}
	return nil
}

func validateJEVAnswer(q models.JEVQuestion, a models.JEVAnswer) error {
	if a.Type != q.Type {
		return fmt.Errorf("type %q does not match question type %q", a.Type, q.Type)
	}
	switch q.Type {
	case models.JEVQuestionTypeNoul:
		if a.Noul == nil {
			return fmt.Errorf("noul is missing")
		}
		return checkUnit("noul", *a.Noul)

	case models.JEVQuestionTypeChoice:
		if a.Choice == nil {
			return fmt.Errorf("choice is missing")
		}
		if _, ok := q.ChoiceCriteria[*a.Choice]; !ok {
			return fmt.Errorf("choice %q is not one of the question options", *a.Choice)
		}
		expected := make(map[string]struct{}, len(q.ChoiceCriteria))
		for opt := range q.ChoiceCriteria {
			expected[opt] = struct{}{}
		}
		if err := checkDistribution(a.Probabilities, expected); err != nil {
			return err
		}
		if err := checkArgmax(*a.Choice, a.Probabilities); err != nil {
			return err
		}
		return checkConfidence(a.Confidence)

	case models.JEVQuestionTypeScore:
		if a.Score == nil {
			return fmt.Errorf("score is missing")
		}
		n := len(q.ScoreCriteria)
		maxLevel := float64(n - 1)
		if math.IsNaN(*a.Score) || math.IsInf(*a.Score, 0) || *a.Score < 0 || *a.Score > maxLevel {
			return fmt.Errorf("score %v is outside level range [0, %d]", *a.Score, n-1)
		}
		expected := make(map[string]struct{}, n)
		for i := 0; i < n; i++ {
			expected[strconv.Itoa(i)] = struct{}{}
		}
		if len(a.Legend) != n {
			return fmt.Errorf("legend has %d levels, question has %d", len(a.Legend), n)
		}
		for key := range a.Legend {
			if _, ok := expected[key]; !ok {
				return fmt.Errorf("legend level %q is not a question level", key)
			}
		}
		if err := checkDistribution(a.Probabilities, expected); err != nil {
			return err
		}
		var expectation float64
		for key, p := range a.Probabilities {
			level, _ := strconv.Atoi(key)
			expectation += float64(level) * p
		}
		if diff := math.Abs(expectation - *a.Score); diff > JEVProbabilityTolerance*maxLevel+jevArgmaxEpsilon {
			return fmt.Errorf("score %v differs from probability-weighted level %v", *a.Score, expectation)
		}
		return checkConfidence(a.Confidence)
	}
	return fmt.Errorf("unsupported type %q", q.Type)
}

func checkUnit(name string, v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
		return fmt.Errorf("%s %v is outside [0, 1]", name, v)
	}
	return nil
}

func checkConfidence(c *float64) error {
	if c == nil {
		return fmt.Errorf("confidence is missing")
	}
	return checkUnit("confidence", *c)
}

func checkDistribution(probs map[string]float64, expected map[string]struct{}) error {
	if probs == nil {
		return fmt.Errorf("probabilities are missing")
	}
	for _, key := range sortedKeys(probs) {
		if _, ok := expected[key]; !ok {
			return fmt.Errorf("probability key %q is not a question option/level", key)
		}
	}
	var sum float64
	for _, key := range sortedKeys(expected) {
		p, ok := probs[key]
		if !ok {
			return fmt.Errorf("probability for %q is missing", key)
		}
		if err := checkUnit("probability for "+strconv.Quote(key), p); err != nil {
			return err
		}
		sum += p
	}
	if math.Abs(sum-1) > JEVProbabilityTolerance {
		return fmt.Errorf("probabilities sum to %v, want 1 ± %v", sum, JEVProbabilityTolerance)
	}
	return nil
}

func checkArgmax(choice string, probs map[string]float64) error {
	chosen := probs[choice]
	for key, p := range probs {
		if p > chosen+jevArgmaxEpsilon {
			return fmt.Errorf("choice %q is not the highest-probability option (%q is higher)", choice, key)
		}
	}
	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
