package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/nopaalh/Relio/backend/controllers"
	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
)

// DATA UJI route stubs; no database, ETL, or external services are involved.
type contextRouteReader struct {
	calls   []string
	id      string
	asOf    models.Date
	dealErr error
}

func (s *contextRouteReader) Snapshot(date models.Date) (models.SnapshotContext, error) {
	s.asOf = date
	return repository.NewSnapshotContext(date, "DATA-UJI", models.AccessScope{
		AllowedAccountIDs: []string{"TEST-ACCOUNT"}, AllowedDealIDs: []string{"TEST-DEAL"},
	})
}

func routeContextMeta(snapshot models.SnapshotContext) models.Meta {
	return models.Meta{AsOf: snapshot.AsOf, MaxAsOf: repository.MaxAsOf, CalendarZone: snapshot.CalendarZone,
		DatasetVersion: snapshot.DatasetVersion, ContractVersion: snapshot.ContractVersion, ContextID: snapshot.ContextID,
		DataState: "partial", Limitations: []string{"DATA UJI"}, Unknowns: []models.Unknown{}}
}

func (s *contextRouteReader) List(_ context.Context, snapshot models.SnapshotContext, _ models.DealListOptions) (models.DealFactsPage, error) {
	s.calls = append(s.calls, "list")
	return models.DealFactsPage{Meta: routeContextMeta(snapshot), Bounds: models.BoundInfo{TruncationReasons: []string{}}, Items: []models.DealFacts{}}, nil
}

func (s *contextRouteReader) Deal(_ context.Context, id string, snapshot models.SnapshotContext) (models.DealFactsResult, error) {
	s.calls, s.id = append(s.calls, "deal"), id
	return models.DealFactsResult{Meta: routeContextMeta(snapshot), Deal: models.DealFacts{DealID: id}}, s.dealErr
}

func (s *contextRouteReader) Graph(_ context.Context, snapshotID string, snapshot models.SnapshotContext, _ models.GraphOptions) (models.GraphResult, error) {
	s.calls, s.id = append(s.calls, "graph"), snapshotID
	return models.GraphResult{Meta: routeContextMeta(snapshot), Bounds: models.BoundInfo{TruncationReasons: []string{}}, Nodes: []models.Node{}, Edges: []models.Edge{}}, nil
}

func (s *contextRouteReader) Timeline(_ context.Context, id string, snapshot models.SnapshotContext, _ models.TimelineOptions) (models.TimelineResult, error) {
	s.calls, s.id = append(s.calls, "timeline"), id
	return models.TimelineResult{Meta: routeContextMeta(snapshot), Bounds: models.BoundInfo{TruncationReasons: []string{}}, Events: []models.Event{}}, nil
}

func (s *contextRouteReader) Evidence(_ context.Context, id string, snapshot models.SnapshotContext) (models.EvidenceResult, error) {
	s.calls, s.id = append(s.calls, "evidence"), id
	return models.EvidenceResult{Meta: routeContextMeta(snapshot), Evidence: models.Evidence{EvidenceID: id}}, nil
}

type contextRouteHealth struct{}

func (contextRouteHealth) Status() models.HealthResponse {
	return models.HealthResponse{Status: "ok", Service: "relio-api"}
}

func newContextRouteTestHandler(reader controllers.ContextReader) http.Handler {
	return NewContextHandler(controllers.NewHealthController(contextRouteHealth{}), controllers.NewContextController(reader))
}

func TestContextRoutesHealthAlias(t *testing.T) {
	handler := newContextRouteTestHandler(nil)
	var first string
	for _, path := range []string{"/healthz", "/api/health"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("liveness requires no data adapter: %d %s", rec.Code, rec.Body.String())
		}
		var body map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || len(body) != 2 || body["status"] != "ok" || body["service"] != "relio-api" {
			t.Fatalf("health is not liveness-only: %s", rec.Body.String())
		}
		if first != "" && rec.Body.String() != first {
			t.Fatal("health alias changed liveness response")
		}
		first = rec.Body.String()
		if path == "/api/health" && rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("API health alias is cacheable")
		}
	}
}

func TestContextRoutesCanonicalWiring(t *testing.T) {
	cases := []struct {
		path, operation, field, id string
	}{
		{"/api/deals", "list", "items", ""},
		{"/api/deals/TEST-DEAL", "deal", "deal", "TEST-DEAL"},
		{"/api/deals/TEST-DEAL/graph", "graph", "nodes", "TEST-DEAL"},
		{"/api/deals/TEST-DEAL/timeline", "timeline", "events", "TEST-DEAL"},
		{"/api/evidence/TEST-EVIDENCE", "evidence", "evidence", "TEST-EVIDENCE"},
		{"/api/evidence/TEST%3AEVIDENCE", "evidence", "evidence", "TEST:EVIDENCE"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			stub, rec := &contextRouteReader{}, httptest.NewRecorder()
			newContextRouteTestHandler(stub).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path+"?as_of=2024-02-29", nil))
			if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/json" || rec.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("route response = %d %v %s", rec.Code, rec.Header(), rec.Body.String())
			}
			if !reflect.DeepEqual(stub.calls, []string{tc.operation}) || stub.id != tc.id || stub.asOf != "2024-02-29" {
				t.Fatalf("wrong reader/path/snapshot: %+v", stub)
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["meta"] == nil || body[tc.field] == nil {
				t.Fatalf("missing canonical envelope: %s", rec.Body.String())
			}
			if tc.operation == "list" || tc.operation == "timeline" || tc.operation == "graph" {
				if string(body[tc.field]) != "[]" || body["bounds"] == nil {
					t.Fatalf("complete empty result is not an initialized bounded page: %s", rec.Body.String())
				}
			}
		})
	}
}

func assertContextRouteError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var body models.ErrorResponse
	if rec.Code != status || rec.Header().Get("Content-Type") != "application/json" || json.Unmarshal(rec.Body.Bytes(), &body) != nil || body.Code != code || body.Message == "" {
		t.Fatalf("expected JSON %d/%s, got %d %v %s", status, code, rec.Code, rec.Header(), rec.Body.String())
	}
	if rec.Header().Get("Location") != "" || strings.Contains(rec.Body.String(), "SECRET") {
		t.Fatalf("route error redirected/leaked path data: %v %s", rec.Header(), rec.Body.String())
	}
}

func TestContextRoutesJSON404And405(t *testing.T) {
	handler := newContextRouteTestHandler(nil)
	for _, path := range []string{
		"/", "/api", "/api/SECRET", "/healthz/extra", "/api/deals/", "/api/evidence/", "/api/copilot",
		"/api/deals/TEST-DEAL/unknown", "/api/deals/TEST-DEAL/graph/extra", "/api/evidence/TEST-EVIDENCE/extra",
		"/api//deals", "/api/deals/../health", "/api/deals/TEST-DEAL//graph", "/api/deals/%2e/graph",
	} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			t.Run(method+path, func(t *testing.T) {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
				assertContextRouteError(t, rec, 404, "not_found")
				if strings.HasPrefix(path, "/api") && rec.Header().Get("Cache-Control") != "no-store" {
					t.Fatal("API route error is cacheable")
				}
			})
		}
	}
	getPaths := []string{"/healthz", "/api/health", "/api/deals", "/api/deals/TEST-DEAL", "/api/deals/TEST-DEAL/graph", "/api/deals/TEST-DEAL/timeline", "/api/evidence/TEST-EVIDENCE", "/api/deals/TEST-DEAL/assessment", "/api/deals/TEST-DEAL/action-candidates"}
	for _, path := range getPaths {
		for _, method := range []string{http.MethodPost, http.MethodHead, http.MethodOptions} {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
			assertContextRouteError(t, rec, 405, "method_not_allowed")
			if rec.Header().Get("Allow") != http.MethodGet {
				t.Fatal("GET route has wrong Allow")
			}
		}
	}
	for _, path := range []string{"/api/deals/TEST-DEAL/actions/compare", "/api/copilot/ask"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		assertContextRouteError(t, rec, 405, "method_not_allowed")
		if rec.Header().Get("Allow") != http.MethodPost {
			t.Fatal("POST route has wrong Allow")
		}
	}
}

func TestContextRoutesUnavailableAndAuthorization(t *testing.T) {
	cases := []struct {
		method, path, body, code string
	}{
		{http.MethodGet, "/api/deals/TEST-DEAL/assessment", "", "assessment_unavailable"},
		{http.MethodGet, "/api/deals/TEST-DEAL/action-candidates", "", "action_candidates_unavailable"},
		{http.MethodPost, "/api/deals/TEST-DEAL/actions/compare", `{"as_of":"2024-02-29","action_ids":["TEST-A","TEST-B"]}`, "comparison_unavailable"},
		{http.MethodPost, "/api/copilot/ask", `{"as_of":"2024-02-29","deal_id":"TEST-DEAL","question":"DATA UJI"}`, "copilot_unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			var notFound string
			for _, category := range []repository.ErrorCode{"", repository.NotFound, repository.AccessDenied, repository.NotVisible} {
				stub := &contextRouteReader{}
				if category != "" {
					stub.dealErr = &repository.RepositoryError{Code: category, EntityID: "SECRET"}
				}
				path := tc.path
				if tc.method == http.MethodGet {
					path += "?as_of=2024-02-29"
				}
				r := httptest.NewRequest(tc.method, path, strings.NewReader(tc.body))
				if tc.method == http.MethodPost {
					r.Header.Set("Content-Type", "application/json")
				}
				rec := httptest.NewRecorder()
				newContextRouteTestHandler(stub).ServeHTTP(rec, r)
				status, code := 503, tc.code
				if category != "" {
					status, code = 404, "not_found"
					if category == repository.NotVisible {
						code = "not_visible_at_snapshot"
					}
				}
				assertContextRouteError(t, rec, status, code)
				if !reflect.DeepEqual(stub.calls, []string{"deal"}) || stub.id != "TEST-DEAL" || stub.asOf != "2024-02-29" || rec.Header().Get("Cache-Control") != "no-store" {
					t.Fatalf("optional endpoint bypassed scoped authorization: %+v", stub)
				}
				if category == repository.NotFound {
					notFound = rec.Body.String()
				} else if category == repository.AccessDenied && rec.Body.String() != notFound {
					t.Fatal("access denial revealed existence")
				}
			}
		})
	}
}

func TestContextRoutesValidateBeforeUnavailableAdapter(t *testing.T) {
	handler := newContextRouteTestHandler(nil)
	for _, path := range []string{"/api/deals", "/api/deals/TEST-DEAL", "/api/deals/TEST-DEAL/graph", "/api/deals/TEST-DEAL/timeline", "/api/evidence/TEST-EVIDENCE", "/api/deals/TEST-DEAL/assessment", "/api/deals/TEST-DEAL/action-candidates"} {
		for _, query := range []string{"?as_of=2026-10-02", "?as_of=", "?as_of=2026-10-01&as_of=2026-10-01"} {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path+query, nil))
			assertContextRouteError(t, rec, 400, "invalid_snapshot")
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		assertContextRouteError(t, rec, 503, "data_not_ready")
	}
}
