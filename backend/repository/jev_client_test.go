package repository

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nopaalh/Relio/backend/models"
)

// DATA UJI: every fixture below is synthetic test data. Response bodies mirror
// the documented example responses in https://docs.typesafe.ai/api (same
// field names, value shapes and level keys). They are not Relio deal data
// and not real provider output.

const testJEVKey = "ts-test-SECRET-should-never-leak-0123456789"

func testJEVQuestions() map[string]models.JEVQuestion {
	return map[string]models.JEVQuestion{
		"is_urgent": {
			Type:         models.JEVQuestionTypeNoul,
			Instructions: "Does this convey urgency?",
			NoulCriteria: &models.JEVNoulCriteria{True: "Explicitly time-sensitive", False: "No urgency expressed"},
		},
		"department": {
			Type:         models.JEVQuestionTypeChoice,
			Instructions: "Which team should handle this?",
			ChoiceCriteria: map[string]any{
				"billing":   "Payments, invoicing, refunds",
				"technical": "Bugs, outages, integrations",
				"sales":     nil,
			},
		},
		"frustration": {
			Type:          models.JEVQuestionTypeScore,
			Instructions:  "How frustrated is the customer?",
			ScoreCriteria: []any{"Calm", "Frustrated", "Very angry"},
		},
	}
}

func testJEVRequest() models.JEVRequest {
	return models.JEVRequest{
		State:     "DATA UJI: Help! My payouts have been failing for 3 days.",
		Questions: testJEVQuestions(),
	}
}

const validJEVBody = `{
  "model": "jev-1.13.0",
  "answers": {
    "is_urgent": {"type": "noul", "noul": 0.95},
    "department": {
      "type": "choice",
      "choice": "billing",
      "probabilities": {"billing": 0.88, "technical": 0.12, "sales": 0.0},
      "confidence": 0.81
    },
    "frustration": {
      "type": "score",
      "score": 1.05,
      "legend": {"0": "Calm", "1": "Frustrated", "2": "Very angry"},
      "probabilities": {"0": 0.0, "1": 0.95, "2": 0.05},
      "confidence": 0.92
    }
  },
  "usage": {"input_tokens": 318, "output_tokens": 34}
}`

func newTestJEVClient(t *testing.T, h http.HandlerFunc, mutate ...func(*JEVClientConfig)) *JEVClient {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	cfg := JEVClientConfig{APIKey: testJEVKey, Endpoint: srv.URL + "/v1/systemone", HTTPClient: srv.Client()}
	for _, m := range mutate {
		m(&cfg)
	}
	c, err := NewJEVClient(cfg)
	if err != nil {
		t.Fatalf("NewJEVClient: %v", err)
	}
	return c
}

func respond(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

func assertNoSecret(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	msg := err.Error()
	for _, s := range []string{testJEVKey, "Bearer", "DATA UJI", "payouts"} {
		if strings.Contains(msg, s) {
			t.Fatalf("error leaks %q: %s", s, msg)
		}
	}
}

func TestJEVClientSendsDocumentedRequest(t *testing.T) {
	var got map[string]any
	c := newTestJEVClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/systemone" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if h := r.Header.Get("Authorization"); h != "Bearer "+testJEVKey {
			t.Errorf("Authorization header mismatch")
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode body: %v", err)
		}
		respond(http.StatusOK, validJEVBody)(w, r)
	})

	resp, err := c.Evaluate(context.Background(), testJEVRequest())
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if got["model"] != "jev-latest" {
		t.Errorf("model = %v, want jev-latest", got["model"])
	}
	if got["state"] != "DATA UJI: Help! My payouts have been failing for 3 days." {
		t.Errorf("state = %v", got["state"])
	}
	qs := got["questions"].(map[string]any)
	noul := qs["is_urgent"].(map[string]any)
	if noul["type"] != "noul" || noul["instructions"] != "Does this convey urgency?" {
		t.Errorf("noul question = %v", noul)
	}
	if crit := noul["criteria"].(map[string]any); crit["true"] != "Explicitly time-sensitive" || crit["false"] != "No urgency expressed" {
		t.Errorf("noul criteria = %v", crit)
	}
	choice := qs["department"].(map[string]any)
	crit := choice["criteria"].(map[string]any)
	if choice["type"] != "choice" || len(crit) != 3 || crit["sales"] != nil {
		t.Errorf("choice question = %v", choice)
	}
	if _, ok := crit["sales"]; !ok {
		t.Errorf("null choice description must be sent as null, got %v", crit)
	}
	score := qs["frustration"].(map[string]any)
	levels := score["criteria"].([]any)
	if score["type"] != "score" || len(levels) != 3 || levels[0] != "Calm" || levels[2] != "Very angry" {
		t.Errorf("score question = %v", score)
	}

	if resp.Model != "jev-1.13.0" {
		t.Errorf("resp.Model = %q", resp.Model)
	}
	if *resp.Answers["is_urgent"].Noul != 0.95 {
		t.Errorf("noul answer = %v", *resp.Answers["is_urgent"].Noul)
	}
	if *resp.Answers["department"].Choice != "billing" {
		t.Errorf("choice answer = %v", *resp.Answers["department"].Choice)
	}
	if *resp.Answers["frustration"].Score != 1.05 {
		t.Errorf("score answer = %v (must keep non-integer weighted score)", *resp.Answers["frustration"].Score)
	}
	if resp.Usage == nil || *resp.Usage.InputTokens != 318 || *resp.Usage.OutputTokens != 34 {
		t.Errorf("usage = %+v", resp.Usage)
	}
}

func TestJEVClientNoulOnlyOmitsCriteria(t *testing.T) {
	var got struct {
		Questions map[string]map[string]any `json:"questions"`
	}
	c := newTestJEVClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		respond(http.StatusOK, `{"model":"jev-1.13.0","answers":{"is_urgent":{"type":"noul","noul":0.95}},"usage":{"input_tokens":296,"output_tokens":20}}`)(w, r)
	})
	req := models.JEVRequest{State: "DATA UJI", Questions: map[string]models.JEVQuestion{
		"is_urgent": {Type: models.JEVQuestionTypeNoul, Instructions: "Does this convey urgency?"},
	}}
	if _, err := c.Evaluate(context.Background(), req); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if _, ok := got.Questions["is_urgent"]["criteria"]; ok {
		t.Errorf("optional noul criteria should be omitted when unset")
	}
}

func TestJEVClientInvalidResponses(t *testing.T) {
	replace := func(old, new string) string {
		if !strings.Contains(validJEVBody, old) {
			t.Fatalf("fixture does not contain %q", old)
		}
		return strings.Replace(validJEVBody, old, new, 1)
	}
	cases := map[string]string{
		"malformed JSON":         `{"model": "jev-1.13.0", "answers": `,
		"trailing data":          validJEVBody + `{}`,
		"trailing closing brace": validJEVBody + `}`,
		"trailing closing array": validJEVBody + `]`,
		"trailing scalar":        validJEVBody + `null`,
		"missing model":          replace(`"model": "jev-1.13.0",`, ``),
		"missing usage": replace(`,
  "usage": {"input_tokens": 318, "output_tokens": 34}`, ``),
		"negative tokens":        replace(`"input_tokens": 318`, `"input_tokens": -1`),
		"missing answer id":      replace(`"is_urgent": {"type": "noul", "noul": 0.95},`, ``),
		"unknown answer id":      replace(`"is_urgent": {`, `"extra": {"type": "noul", "noul": 0.5}, "is_urgent": {`),
		"type mismatch":          replace(`{"type": "noul", "noul": 0.95}`, `{"type": "score", "noul": 0.95}`),
		"noul missing":           replace(`"noul": 0.95`, `"noul": null`),
		"noul above 1":           replace(`"noul": 0.95`, `"noul": 1.2`),
		"noul negative":          replace(`"noul": 0.95`, `"noul": -0.1`),
		"noul not a number":      replace(`"noul": 0.95`, `"noul": "high"`),
		"choice outside options": replace(`"choice": "billing"`, `"choice": "legal"`),
		"choice not argmax":      replace(`"choice": "billing"`, `"choice": "technical"`),
		"choice prob missing":    replace(`"billing": 0.88, "technical": 0.12, "sales": 0.0`, `"billing": 0.88, "technical": 0.12`),
		"choice prob unknown":    replace(`"sales": 0.0}`, `"sales": 0.0, "legal": 0.0}`),
		"choice prob null":       replace(`"sales": 0.0`, `"sales": null`),
		"choice sum low":         replace(`"technical": 0.12`, `"technical": 0.02`),
		"choice sum high":        replace(`"technical": 0.12`, `"technical": 0.22`),
		"choice prob negative":   replace(`"billing": 0.88, "technical": 0.12, "sales": 0.0`, `"billing": 1.1, "technical": 0.0, "sales": -0.1`),
		"choice no confidence":   replace(`"confidence": 0.81`, `"confidence": null`),
		"confidence above 1":     replace(`"confidence": 0.81`, `"confidence": 1.5`),
		"score above range":      replace(`"score": 1.05`, `"score": 2.5`),
		"score negative":         replace(`"score": 1.05`, `"score": -0.1`),
		"score not expectation":  replace(`"score": 1.05`, `"score": 1.6`),
		"legend level unknown":   replace(`"2": "Very angry"}`, `"3": "Very angry"}`),
		"legend missing":         replace(`"legend": {"0": "Calm", "1": "Frustrated", "2": "Very angry"},`, ``),
		"legend value null":      replace(`"0": "Calm"`, `"0": null`),
		"score prob null":        replace(`"0": 0.0`, `"0": null`),
		"score level missing":    replace(`"probabilities": {"0": 0.0, "1": 0.95, "2": 0.05}`, `"probabilities": {"0": 0.05, "1": 0.95}`),
		"score sum off":          replace(`"2": 0.05}`, `"2": 0.5}`),
		"score no confidence":    replace(`"confidence": 0.92`, `"confidence": null`),
		"answers missing":        `{"model":"jev-1.13.0","usage":{}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			c := newTestJEVClient(t, respond(http.StatusOK, body))
			_, err := c.Evaluate(context.Background(), testJEVRequest())
			if !errors.Is(err, ErrJEVInvalidResponse) {
				t.Fatalf("err = %v, want ErrJEVInvalidResponse", err)
			}
			assertNoSecret(t, err)
		})
	}
}

func TestValidateJEVResponseProbabilityTolerance(t *testing.T) {
	qs := map[string]models.JEVQuestion{"department": testJEVQuestions()["department"]}
	mk := func(technical float64) models.JEVResponse {
		choice, conf := "billing", 0.8
		return models.JEVResponse{
			Model: "jev-1.13.0",
			Usage: &models.JEVUsage{},
			Answers: map[string]models.JEVAnswer{"department": {
				Type: "choice", Choice: &choice, Confidence: &conf,
				Probabilities: map[string]float64{"billing": 0.8, "technical": technical, "sales": 0},
			}},
		}
	}
	// Sum = 0.8 + technical. Inside ±0.02 passes; outside fails.
	for _, tc := range []struct {
		technical float64
		ok        bool
	}{{0.2, true}, {0.185, true}, {0.215, true}, {0.17, false}, {0.23, false}} {
		err := ValidateJEVResponse(qs, mk(tc.technical))
		if (err == nil) != tc.ok {
			t.Errorf("technical=%v: err=%v, want ok=%v", tc.technical, err, tc.ok)
		}
	}
}

func TestValidateJEVResponseScoreAllowsBetweenLevels(t *testing.T) {
	qs := map[string]models.JEVQuestion{"frustration": testJEVQuestions()["frustration"]}
	score, conf := 1.5, 0.3
	resp := models.JEVResponse{
		Model: "jev-1.13.0",
		Usage: &models.JEVUsage{},
		Answers: map[string]models.JEVAnswer{"frustration": {
			Type: "score", Score: &score, Confidence: &conf,
			Legend:        map[string]string{"0": "Calm", "1": "Frustrated", "2": "Very angry"},
			Probabilities: map[string]float64{"0": 0.0, "1": 0.5, "2": 0.5},
		}},
	}
	if err := ValidateJEVResponse(qs, resp); err != nil {
		t.Fatalf("weighted score between levels should be valid: %v", err)
	}
}

func TestJEVClientStatusClassification(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusBadRequest, ErrJEVInvalidRequest},
		{http.StatusUnprocessableEntity, ErrJEVInvalidRequest},
		{http.StatusUnauthorized, ErrJEVAuthentication},
		{http.StatusForbidden, ErrJEVAuthentication},
		{http.StatusTooManyRequests, ErrJEVRateLimited},
		{http.StatusInternalServerError, ErrJEVProviderFailure},
		{http.StatusBadGateway, ErrJEVProviderFailure},
		{529, ErrJEVProviderFailure},
		{http.StatusNotFound, ErrJEVProviderFailure},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status)+"_"+itoa(tc.status), func(t *testing.T) {
			calls := 0
			// The error body echoes the key and state to prove neither leaks.
			c := newTestJEVClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Retry-After", "7")
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, `{"error":"bad key `+testJEVKey+` for DATA UJI payouts"}`)
			})
			_, err := c.Evaluate(context.Background(), testJEVRequest())
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			var je *JEVError
			if !errors.As(err, &je) || je.StatusCode != tc.status {
				t.Fatalf("JEVError status mismatch: %#v", err)
			}
			if tc.status == http.StatusTooManyRequests && je.RetryAfter != "7" {
				t.Errorf("RetryAfter = %q", je.RetryAfter)
			}
			if calls != 1 {
				t.Errorf("provider called %d times, want exactly 1 (no automatic retry)", calls)
			}
			assertNoSecret(t, err)
		})
	}
}

func TestJEVClientRejectsInvalidRequestWithoutCalling(t *testing.T) {
	calls := 0
	c := newTestJEVClient(t, func(w http.ResponseWriter, r *http.Request) { calls++ })
	bad := map[string]models.JEVRequest{
		"no state":     {Questions: testJEVQuestions()},
		"empty state":  {State: "  ", Questions: testJEVQuestions()},
		"no questions": {State: "DATA UJI"},
		"unknown type": {State: "DATA UJI", Questions: map[string]models.JEVQuestion{"q": {Type: "bool", Instructions: "x"}}},
		"empty instr":  {State: "DATA UJI", Questions: map[string]models.JEVQuestion{"q": {Type: "noul", Instructions: ""}}},
		"one option":   {State: "DATA UJI", Questions: map[string]models.JEVQuestion{"q": {Type: "choice", Instructions: "x", ChoiceCriteria: map[string]any{"a": nil}}}},
		"one level":    {State: "DATA UJI", Questions: map[string]models.JEVQuestion{"q": {Type: "score", Instructions: "x", ScoreCriteria: []any{"a"}}}},
		"11 levels":    {State: "DATA UJI", Questions: map[string]models.JEVQuestion{"q": {Type: "score", Instructions: "x", ScoreCriteria: make([]any, 11)}}},
		"mixed crit":   {State: "DATA UJI", Questions: map[string]models.JEVQuestion{"q": {Type: "noul", Instructions: "x", ScoreCriteria: []any{"a", "b"}}}},
	}
	for name, req := range bad {
		t.Run(name, func(t *testing.T) {
			_, err := c.Evaluate(context.Background(), req)
			if !errors.Is(err, ErrJEVInvalidRequest) {
				t.Fatalf("err = %v, want ErrJEVInvalidRequest", err)
			}
		})
	}
	if calls != 0 {
		t.Fatalf("provider called %d times for invalid local requests", calls)
	}
}

func TestJEVClientTimeoutAndCancellation(t *testing.T) {
	newSlowClient := func(st *testing.T, mutate ...func(*JEVClientConfig)) *JEVClient {
		st.Helper()
		done := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-done:
			case <-r.Context().Done():
			}
		}))
		st.Cleanup(func() {
			close(done)
			srv.Close()
		})
		cfg := JEVClientConfig{APIKey: testJEVKey, Endpoint: srv.URL + "/v1/systemone", HTTPClient: srv.Client()}
		for _, m := range mutate {
			m(&cfg)
		}
		c, err := NewJEVClient(cfg)
		if err != nil {
			st.Fatalf("NewJEVClient: %v", err)
		}
		return c
	}

	t.Run("client timeout", func(t *testing.T) {
		c := newSlowClient(t, func(cfg *JEVClientConfig) { cfg.Timeout = 50 * time.Millisecond })
		_, err := c.Evaluate(context.Background(), testJEVRequest())
		if !errors.Is(err, ErrJEVTransport) || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v, want transport + DeadlineExceeded", err)
		}
		assertNoSecret(t, err)
	})
	t.Run("caller deadline", func(t *testing.T) {
		c := newSlowClient(t)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		_, err := c.Evaluate(ctx, testJEVRequest())
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v, want DeadlineExceeded", err)
		}
	})
	t.Run("caller cancel", func(t *testing.T) {
		c := newSlowClient(t)
		ctx, cancel := context.WithCancel(context.Background())
		go func() { time.Sleep(30 * time.Millisecond); cancel() }()
		_, err := c.Evaluate(ctx, testJEVRequest())
		if !errors.Is(err, ErrJEVTransport) || !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want transport + Canceled", err)
		}
	})
}

func TestJEVClientTransportFailure(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	endpoint := srv.URL + "/v1/systemone"
	srv.Close() // connection refused
	c, err := NewJEVClient(JEVClientConfig{APIKey: testJEVKey, Endpoint: endpoint})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Evaluate(context.Background(), testJEVRequest())
	if !errors.Is(err, ErrJEVTransport) {
		t.Fatalf("err = %v, want ErrJEVTransport", err)
	}
	assertNoSecret(t, err)
}

func TestJEVClientResponseTooLarge(t *testing.T) {
	big := `{"model":"jev-1.13.0","pad":"` + strings.Repeat("x", 4096) + `"}`
	c := newTestJEVClient(t, respond(http.StatusOK, big), func(cfg *JEVClientConfig) { cfg.MaxResponseBytes = 1024 })
	_, err := c.Evaluate(context.Background(), testJEVRequest())
	if !errors.Is(err, ErrJEVInvalidResponse) || !strings.Contains(err.Error(), "exceeds 1024 bytes") {
		t.Fatalf("err = %v, want size-limit invalid response", err)
	}
}

func TestNewJEVClientConfig(t *testing.T) {
	if _, err := NewJEVClient(JEVClientConfig{}); !errors.Is(err, ErrJEVNotConfigured) {
		t.Errorf("empty key: err = %v", err)
	}
	if _, err := NewJEVClient(JEVClientConfig{APIKey: testJEVKey, MaxResponseBytes: 1<<63 - 1}); !errors.Is(err, ErrJEVNotConfigured) {
		t.Errorf("overflowing response limit: err = %v", err)
	}
	for _, ep := range []string{"http://api.typesafe.ai/v1/systemone", "ftp://x/y", "not a url", "https://user:pw@api.typesafe.ai/v1/systemone"} {
		if _, err := NewJEVClient(JEVClientConfig{APIKey: testJEVKey, Endpoint: ep}); !errors.Is(err, ErrJEVNotConfigured) {
			t.Errorf("endpoint %q: err = %v, want ErrJEVNotConfigured", ep, err)
		} else {
			assertNoSecret(t, err)
		}
	}
	c, err := NewJEVClient(JEVClientConfig{APIKey: testJEVKey})
	if err != nil {
		t.Fatal(err)
	}
	if c.endpoint != JEVDefaultEndpoint || c.model != models.JEVModelLatest || c.timeout != JEVDefaultTimeout || c.maxBytes != JEVDefaultMaxResponseBytes {
		t.Errorf("defaults = %+v", c)
	}
	if c.http.Timeout <= 0 {
		t.Errorf("default HTTP client must have a finite timeout")
	}
}

func itoa(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}
