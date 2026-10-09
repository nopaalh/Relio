package repository

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/nopaalh/Relio/backend/models"
)

// TestJEVLive is an opt-in POC against the real TypeSafe endpoint. It makes
// at most ONE paid request and never retries. It only runs when:
//
//	RUN_JEV_LIVE=1          (explicit permission for network + paid call)
//	TYPESAFE_API_KEY=...    (read from the local environment, never logged)
//
// Otherwise it is reported as SKIP, which is not a passing integration.
//
// The state is synthetic DATA UJI. Answers prove connectivity, the documented
// schema and the validator; they are not a Relio deal assessment and are not
// expected to equal any particular value.
func TestJEVLive(t *testing.T) {
	if os.Getenv("RUN_JEV_LIVE") != "1" {
		t.Skip("SKIP: RUN_JEV_LIVE != 1 (live JEV call not permitted)")
	}
	key := os.Getenv("TYPESAFE_API_KEY")
	if key == "" {
		t.Skip("SKIP: TYPESAFE_API_KEY is not set in the environment")
	}

	client, err := NewJEVClient(JEVClientConfig{APIKey: key, Timeout: 60 * time.Second})
	if err != nil {
		t.Fatalf("NewJEVClient: %v", err)
	}

	req := models.JEVRequest{
		Model: models.JEVModelLatest,
		State: map[string]any{
			"label": "DATA UJI - synthetic, not a Relio deal",
			"note":  "Customer wrote: our invoices failed twice this week and we need this fixed before Friday.",
		},
		Questions: map[string]models.JEVQuestion{
			"poc_noul": {
				Type:         models.JEVQuestionTypeNoul,
				Instructions: "Does `note` state a deadline?",
			},
			"poc_choice": {
				Type:         models.JEVQuestionTypeChoice,
				Instructions: "Which topic is `note` mainly about?",
				ChoiceCriteria: map[string]any{
					"billing":   "Payments or invoices",
					"technical": "Bugs or integrations",
					"other":     nil,
				},
			},
			"poc_score": {
				Type:          models.JEVQuestionTypeScore,
				Instructions:  "How urgent does `note` sound?",
				ScoreCriteria: []any{"Not urgent", "Somewhat urgent", "Very urgent"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	start := time.Now()
	resp, err := client.Evaluate(ctx, req)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("live JEV call failed after %s: %v", elapsed.Round(time.Millisecond), err)
	}

	t.Logf("LIVE JEV OK in %s; provider model=%q", elapsed.Round(time.Millisecond), resp.Model)
	if u := resp.Usage; u != nil {
		t.Logf("usage input_tokens=%s output_tokens=%s", fmtTokens(u.InputTokens), fmtTokens(u.OutputTokens))
	}
	ids := make([]string, 0, len(resp.Answers))
	for id := range resp.Answers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		a := resp.Answers[id]
		switch a.Type {
		case models.JEVQuestionTypeNoul:
			t.Logf("DATA UJI %s noul=%v", id, *a.Noul)
		case models.JEVQuestionTypeChoice:
			t.Logf("DATA UJI %s choice=%q probabilities=%v confidence=%v", id, *a.Choice, a.Probabilities, *a.Confidence)
		case models.JEVQuestionTypeScore:
			t.Logf("DATA UJI %s score=%v probabilities=%v confidence=%v", id, *a.Score, a.Probabilities, *a.Confidence)
		}
	}
}

func fmtTokens(v *int64) string {
	if v == nil {
		return "absent"
	}
	b := []byte{}
	n := *v
	if n == 0 {
		return "0"
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
