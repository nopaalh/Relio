package repository

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/nopaalh/Relio/backend/models"
)

func TestJEVClientBlocksRedirects(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		for _, customPolicy := range []bool{false, true} {
			t.Run(strconv.Itoa(status)+"/custom_policy="+strconv.FormatBool(customPolicy), func(t *testing.T) {
				var sourceCalls, targetCalls, policyCalls atomic.Int32
				target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					targetCalls.Add(1)
					respond(http.StatusOK, validJEVBody)(w, r)
				}))
				defer target.Close()
				source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					sourceCalls.Add(1)
					w.Header().Set("Location", target.URL)
					w.WriteHeader(status)
				}))
				defer source.Close()
				injected := source.Client()
				if customPolicy {
					injected.CheckRedirect = func(*http.Request, []*http.Request) error {
						policyCalls.Add(1)
						return nil
					}
				}
				c, err := NewJEVClient(JEVClientConfig{APIKey: testJEVKey, Endpoint: source.URL, HTTPClient: injected})
				if err != nil {
					t.Fatal(err)
				}
				_, err = c.Evaluate(context.Background(), testJEVRequest())
				if !errors.Is(err, ErrJEVProviderFailure) {
					t.Errorf("err = %v, want provider failure for redirect", err)
				}
				var providerErr *JEVError
				if !errors.As(err, &providerErr) || providerErr.StatusCode != status {
					t.Errorf("redirect status was not preserved")
				}
				if sourceCalls.Load() != 1 || targetCalls.Load() != 0 || policyCalls.Load() != 0 {
					t.Errorf("calls source=%d target=%d injected-policy=%d; want 1/0/0", sourceCalls.Load(), targetCalls.Load(), policyCalls.Load())
				}
				if c.http == injected {
					t.Error("client must own a copy of the injected http.Client")
				}
				if (injected.CheckRedirect != nil) != customPolicy {
					t.Fatal("constructor mutated the injected redirect policy")
				}
				if customPolicy {
					before := policyCalls.Load()
					if err := injected.CheckRedirect(nil, nil); err != nil || policyCalls.Load() != before+1 {
						t.Error("injected redirect callback was replaced")
					}
				}
				assertNoSecret(t, err)
			})
		}
	}
}

func TestJEVClientAcceptsTrailingWhitespace(t *testing.T) {
	c := newTestJEVClient(t, respond(http.StatusOK, validJEVBody+" \n\t\r"))
	if _, err := c.Evaluate(context.Background(), testJEVRequest()); err != nil {
		t.Fatalf("valid JSON with trailing whitespace: %v", err)
	}
}

func TestJEVClientValidatesSerializedContentBeforeCalling(t *testing.T) {
	var calls atomic.Int32
	c := newTestJEVClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		respond(http.StatusOK, validJEVBody)(w, r)
	})
	var nilMap map[string]any
	var nilSlice []any
	var nilPointer *string
	values := []struct {
		name  string
		value any
		null  bool
	}{
		{"boolean", true, false},
		{"number", 7, false},
		{"raw boolean", json.RawMessage(`false`), false},
		{"raw number", json.RawMessage(`7`), false},
		{"raw null", json.RawMessage(`null`), true},
		{"nil map", nilMap, true},
		{"nil slice", nilSlice, true},
		{"nil pointer", nilPointer, true},
		{"invalid JSON", json.RawMessage(`{"broken":`), false},
		{"non JSON", make(chan int), false},
		{"non JSON nested", map[string]any{"value": func() {}}, false},
	}
	targets := []struct {
		name       string
		questionID string
		set        func(*models.JEVRequest, *models.JEVQuestion, any)
		allowNull  bool
	}{
		{"state", "is_urgent", func(r *models.JEVRequest, _ *models.JEVQuestion, v any) { r.State = v }, false},
		{"instructions", "is_urgent", func(_ *models.JEVRequest, q *models.JEVQuestion, v any) { q.Instructions = v }, false},
		{"noul true", "is_urgent", func(_ *models.JEVRequest, q *models.JEVQuestion, v any) { q.NoulCriteria.True = v }, false},
		{"noul false", "is_urgent", func(_ *models.JEVRequest, q *models.JEVQuestion, v any) { q.NoulCriteria.False = v }, false},
		{"choice description", "department", func(_ *models.JEVRequest, q *models.JEVQuestion, v any) { q.ChoiceCriteria["sales"] = v }, true},
		{"score level", "frustration", func(_ *models.JEVRequest, q *models.JEVQuestion, v any) { q.ScoreCriteria[0] = v }, false},
	}
	for _, target := range targets {
		for _, value := range values {
			if target.allowNull && value.null {
				continue
			}
			t.Run(target.name+"/"+value.name, func(t *testing.T) {
				req := testJEVRequest()
				q := req.Questions[target.questionID]
				target.set(&req, &q, value.value)
				req.Questions[target.questionID] = q
				before := calls.Load()
				_, err := c.Evaluate(context.Background(), req)
				if !errors.Is(err, ErrJEVInvalidRequest) {
					t.Errorf("err = %v, want locally invalid request", err)
				}
				if calls.Load() != before {
					t.Error("invalid input reached the provider")
				}
				assertNoSecret(t, err)
			})
		}
	}
	blank := " \t"
	for _, value := range []any{&blank, json.RawMessage(`"  "`)} {
		for _, instructions := range []bool{false, true} {
			req := testJEVRequest()
			if instructions {
				q := req.Questions["is_urgent"]
				q.Instructions = value
				req.Questions["is_urgent"] = q
			} else {
				req.State = value
			}
			before := calls.Load()
			_, err := c.Evaluate(context.Background(), req)
			if !errors.Is(err, ErrJEVInvalidRequest) || calls.Load() != before {
				t.Errorf("blank serialized content must fail locally: %v", err)
			}
		}
	}
}

func TestJEVClientAcceptsStructuredContentAndNullChoiceDescription(t *testing.T) {
	values := []any{
		"DATA UJI",
		map[string]any{"nested": []any{true, 0, nil}},
		[]any{"DATA UJI", false, 0, nil},
		struct {
			Count int `json:"count"`
		}{Count: 0},
		json.RawMessage(`{"nested":false}`),
		json.RawMessage(`[1,true,null]`),
		json.RawMessage(`"DATA UJI"`),
	}
	c := newTestJEVClient(t, respond(http.StatusOK, validJEVBody))
	for i, value := range values {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			req := testJEVRequest()
			req.State = value
			q := req.Questions["is_urgent"]
			q.Instructions = value
			q.NoulCriteria = &models.JEVNoulCriteria{True: value}
			req.Questions["is_urgent"] = q
			q = req.Questions["department"]
			q.ChoiceCriteria["billing"] = value
			q.ChoiceCriteria["sales"] = json.RawMessage(`null`)
			req.Questions["department"] = q
			q = req.Questions["frustration"]
			q.ScoreCriteria = []any{value, "", value}
			req.Questions["frustration"] = q
			if _, err := c.Evaluate(context.Background(), req); err != nil {
				t.Fatalf("provider-allowed structured content was rejected: %v", err)
			}
		})
	}
}

func TestJEVClientRedactsValidationErrors(t *testing.T) {
	secret := testJEVKey + " DATA UJI payouts"
	bodies := map[string]string{
		"answer id":       strings.Replace(validJEVBody, `"is_urgent":`, `"`+secret+`":`, 1),
		"answer type":     strings.Replace(validJEVBody, `"type": "noul"`, `"type": "`+secret+`"`, 1),
		"choice":          strings.Replace(validJEVBody, `"choice": "billing"`, `"choice": "`+secret+`"`, 1),
		"probability key": strings.Replace(validJEVBody, `"sales": 0.0`, `"`+secret+`": 0.0`, 1),
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			c := newTestJEVClient(t, respond(http.StatusOK, body))
			_, err := c.Evaluate(context.Background(), testJEVRequest())
			if !errors.Is(err, ErrJEVInvalidResponse) {
				t.Fatalf("err = %v, want invalid response", err)
			}
			assertNoSecret(t, err)
		})
	}
	t.Run("request", func(t *testing.T) {
		c := newTestJEVClient(t, func(http.ResponseWriter, *http.Request) { t.Error("invalid request reached the provider") })
		req := models.JEVRequest{State: "DATA UJI", Questions: map[string]models.JEVQuestion{
			secret: {Type: "unsupported", Instructions: "DATA UJI"},
		}}
		_, err := c.Evaluate(context.Background(), req)
		if !errors.Is(err, ErrJEVInvalidRequest) {
			t.Fatalf("err = %v, want invalid request", err)
		}
		assertNoSecret(t, err)
	})
}

func TestJEVErrorDoesNotRenderCause(t *testing.T) {
	cause := errors.New(testJEVKey + " DATA UJI payouts")
	err := &JEVError{Kind: ErrJEVTransport, Err: cause}
	assertNoSecret(t, err)
	if !errors.Is(err, ErrJEVTransport) || !errors.Is(err, cause) {
		t.Fatal("error classification or underlying cause was lost")
	}
}
