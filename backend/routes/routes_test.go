package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nopaalh/Relio/backend/controllers"
	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/services"
)

func TestHealthEndpoint(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	testHandler(services.NewDealService(nil)).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body models.HealthResponse
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "ok" || body.Service != "relio-api" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestHealthEndpointRejectsNonGet(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/healthz", nil)

	testHandler(services.NewDealService(nil)).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	if got := recorder.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("Allow = %q, want %q", got, http.MethodGet)
	}
}

func TestDealRouteWiring(t *testing.T) {
	reader := &routeDealReader{}
	handler := testHandler(reader)

	listRecorder := httptest.NewRecorder()
	handler.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/api/deals", nil))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRecorder.Code, http.StatusOK)
	}
	var deals []models.Deal
	if err := json.NewDecoder(listRecorder.Body).Decode(&deals); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(deals) != 1 || deals[0].ID != "DL-001" {
		t.Fatalf("unexpected list: %+v", deals)
	}

	detailRecorder := httptest.NewRecorder()
	handler.ServeHTTP(detailRecorder, httptest.NewRequest(http.MethodGet, "/api/deals/DL-002", nil))
	if detailRecorder.Code != http.StatusOK {
		t.Fatalf("detail status = %d, want %d", detailRecorder.Code, http.StatusOK)
	}
	var deal models.Deal
	if err := json.NewDecoder(detailRecorder.Body).Decode(&deal); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if deal.ID != "DL-002" || reader.lastID != "DL-002" {
		t.Fatalf("unexpected detail: deal=%+v, reader ID=%q", deal, reader.lastID)
	}
}

func TestDealRoutesWithoutAdapter(t *testing.T) {
	handler := testHandler(services.NewDealService(nil))
	for _, path := range []string{"/api/deals", "/api/deals/DL-001"} {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
			}
			var body models.ErrorResponse
			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if body.Code != "deal_data_unavailable" {
				t.Fatalf("unexpected error: %+v", body)
			}
		})
	}
}

func TestDealRoutesRejectNestedPath(t *testing.T) {
	recorder := httptest.NewRecorder()
	testHandler(services.NewDealService(nil)).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/deals/DL-001/graph", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func testHandler(deals controllers.DealReader) http.Handler {
	return NewHandler(
		controllers.NewHealthController(services.NewHealthService()),
		controllers.NewDealController(deals),
	)
}

type routeDealReader struct {
	lastID string
}

func (r *routeDealReader) List(ctx context.Context, asOf time.Time) ([]models.Deal, error) {
	return []models.Deal{{ID: "DL-001"}}, nil
}

func (r *routeDealReader) FindByID(ctx context.Context, dealID string, asOf time.Time) (models.Deal, error) {
	r.lastID = dealID
	return models.Deal{ID: dealID}, nil
}
