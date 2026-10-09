package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
)

type stubDealReader struct {
	deals     []models.Deal
	deal      models.Deal
	err       error
	listCalls int
	findCalls int
	lastID    string
	lastAsOf  time.Time
	lastCtx   context.Context
}

func (s *stubDealReader) List(ctx context.Context, asOf time.Time) ([]models.Deal, error) {
	s.listCalls++
	s.lastCtx, s.lastAsOf = ctx, asOf
	return s.deals, s.err
}

func (s *stubDealReader) FindByID(ctx context.Context, dealID string, asOf time.Time) (models.Deal, error) {
	s.findCalls++
	s.lastCtx, s.lastID, s.lastAsOf = ctx, dealID, asOf
	return s.deal, s.err
}

func TestDealControllerList(t *testing.T) {
	reader := &stubDealReader{deals: []models.Deal{{ID: "DL-001", AccountID: "P01"}}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := httptest.NewRequest(http.MethodGet, "/api/deals?as_of=2026-10-01", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()

	NewDealController(reader).List(recorder, request)

	if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected response: status=%d, headers=%v", recorder.Code, recorder.Header())
	}
	var deals []models.Deal
	if err := json.Unmarshal(recorder.Body.Bytes(), &deals); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(deals) != 1 || deals[0].ID != "DL-001" || deals[0].AccountID != "P01" {
		t.Fatalf("unexpected deals: %+v", deals)
	}
	wantDate := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	if reader.listCalls != 1 || reader.lastCtx != ctx || !reader.lastAsOf.Equal(wantDate) {
		t.Fatalf("unexpected call: count=%d, context=%v, asOf=%v", reader.listCalls, reader.lastCtx, reader.lastAsOf)
	}
}

func TestDealControllerListEmptySuccess(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewDealController(&stubDealReader{}).List(recorder, httptest.NewRequest(http.MethodGet, "/api/deals", nil))
	if recorder.Code != http.StatusOK || strings.TrimSpace(recorder.Body.String()) != "[]" {
		t.Fatalf("unexpected empty response: status=%d, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestDealControllerGet(t *testing.T) {
	reader := &stubDealReader{deal: models.Deal{ID: "DL-002", AccountID: "P02"}}
	request := httptest.NewRequest(http.MethodGet, "/api/deals/DL-002", nil)
	request.SetPathValue("deal_id", " DL-002 ")
	recorder := httptest.NewRecorder()

	NewDealController(reader).Get(recorder, request)

	var deal models.Deal
	if err := json.Unmarshal(recorder.Body.Bytes(), &deal); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	wantDate := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	if recorder.Code != http.StatusOK || deal.ID != "DL-002" || reader.lastID != "DL-002" || reader.findCalls != 1 || !reader.lastAsOf.Equal(wantDate) {
		t.Fatalf("unexpected response/call: status=%d, deal=%+v, reader=%+v", recorder.Code, deal, reader)
	}
}

func TestDealControllerRejectsUnsupportedSnapshot(t *testing.T) {
	queries := []string{
		"?as_of=",
		"?as_of=invalid",
		"?as_of=2026-02-30",
		"?as_of=2026-09-30",
		"?as_of=2026-10-02",
		"?as_of=2026-10-01T00:00:00Z",
		"?as_of=2026-10-01&as_of=2026-10-01",
		"?as_of=%ZZ",
	}
	for _, operation := range []string{"list", "get"} {
		for _, query := range queries {
			t.Run(operation+query, func(t *testing.T) {
				reader := &stubDealReader{}
				controller := NewDealController(reader)
				request := httptest.NewRequest(http.MethodGet, "/api/deals"+query, nil)
				request.SetPathValue("deal_id", "DL-001")
				recorder := httptest.NewRecorder()
				if operation == "list" {
					controller.List(recorder, request)
				} else {
					controller.Get(recorder, request)
				}
				assertDealError(t, recorder, http.StatusBadRequest, "invalid_as_of")
				if reader.listCalls != 0 || reader.findCalls != 0 {
					t.Fatal("invalid snapshot must not reach the data reader")
				}
			})
		}
	}
}

func TestDealControllerRejectsEmptyID(t *testing.T) {
	reader := &stubDealReader{}
	request := httptest.NewRequest(http.MethodGet, "/api/deals/empty", nil)
	request.SetPathValue("deal_id", " ")
	recorder := httptest.NewRecorder()
	NewDealController(reader).Get(recorder, request)
	assertDealError(t, recorder, http.StatusBadRequest, "invalid_deal_id")
	if reader.findCalls != 0 {
		t.Fatal("empty ID must not reach the data reader")
	}
}

func TestDealControllerRejectsNonGet(t *testing.T) {
	for _, operation := range []string{"list", "get"} {
		t.Run(operation, func(t *testing.T) {
			reader := &stubDealReader{}
			controller := NewDealController(reader)
			request := httptest.NewRequest(http.MethodPost, "/api/deals", nil)
			request.SetPathValue("deal_id", "DL-001")
			recorder := httptest.NewRecorder()
			if operation == "list" {
				controller.List(recorder, request)
			} else {
				controller.Get(recorder, request)
			}
			assertDealError(t, recorder, http.StatusMethodNotAllowed, "method_not_allowed")
			if recorder.Header().Get("Allow") != http.MethodGet || reader.listCalls != 0 || reader.findCalls != 0 {
				t.Fatalf("unexpected method handling: headers=%v, reader=%+v", recorder.Header(), reader)
			}
		})
	}
}

func TestDealControllerMapsReaderErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"not found", fmt.Errorf("lookup: %w", repository.ErrDealNotFound), http.StatusNotFound, "deal_not_found"},
		{"unavailable", fmt.Errorf("lookup: %w", repository.ErrDealDataUnavailable), http.StatusServiceUnavailable, "deal_data_unavailable"},
		{"internal", errors.New("private database credential detail"), http.StatusInternalServerError, "internal_error"},
	}
	for _, operation := range []string{"list", "get"} {
		for _, tc := range cases {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				controller := NewDealController(&stubDealReader{err: tc.err})
				request := httptest.NewRequest(http.MethodGet, "/api/deals", nil)
				request.SetPathValue("deal_id", "DL-001")
				recorder := httptest.NewRecorder()
				if operation == "list" {
					controller.List(recorder, request)
				} else {
					controller.Get(recorder, request)
				}
				assertDealError(t, recorder, tc.status, tc.code)
				if strings.Contains(recorder.Body.String(), "credential") {
					t.Fatal("internal repository error must not be exposed")
				}
			})
		}
	}
}

func TestDealControllerRejectsUnencodableData(t *testing.T) {
	reader := &stubDealReader{deal: models.Deal{StageSince: time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)}}
	request := httptest.NewRequest(http.MethodGet, "/api/deals/DL-001", nil)
	request.SetPathValue("deal_id", "DL-001")
	recorder := httptest.NewRecorder()
	NewDealController(reader).Get(recorder, request)
	assertDealError(t, recorder, http.StatusInternalServerError, "internal_error")
}

func assertDealError(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if recorder.Code != status || recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected response: status=%d, headers=%v, body=%s", recorder.Code, recorder.Header(), recorder.Body.String())
	}
	var body models.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Code != code || body.Message == "" {
		t.Fatalf("unexpected error: %+v", body)
	}
}
