package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
)

// DATA UJI only: these stubs make no dataset, database, or provider calls.
type contextHTTPStub struct {
	snapshotErr   error
	readErr       error
	onRead        func(context.Context) error
	snapshotCalls int
	reads         []string
	asOf          models.Date
	snapshot      models.SnapshotContext
	ctx           context.Context
	id            string
	listOptions   models.DealListOptions
	graphOptions  models.GraphOptions
	timeOptions   models.TimelineOptions
}

func (s *contextHTTPStub) Snapshot(asOf models.Date) (models.SnapshotContext, error) {
	s.snapshotCalls++
	s.asOf = asOf
	if s.snapshotErr != nil {
		return models.SnapshotContext{}, s.snapshotErr
	}
	return repository.NewSnapshotContext(asOf, "DATA-UJI", models.AccessScope{
		AllowedAccountIDs: []string{"TEST-ACCOUNT"}, AllowedDealIDs: []string{"TEST-DEAL"},
	})
}

func (s *contextHTTPStub) record(ctx context.Context, operation, id string, snapshot models.SnapshotContext) error {
	s.reads = append(s.reads, operation)
	s.ctx, s.id, s.snapshot = ctx, id, snapshot
	if s.onRead != nil {
		return s.onRead(ctx)
	}
	return s.readErr
}

func contextTestMeta(snapshot models.SnapshotContext) models.Meta {
	return models.Meta{
		ContractVersion: snapshot.ContractVersion, DatasetVersion: snapshot.DatasetVersion,
		AsOf: snapshot.AsOf, MaxAsOf: repository.MaxAsOf, CalendarZone: snapshot.CalendarZone,
		ContextID: snapshot.ContextID, DataState: "partial", Limitations: []string{"DATA UJI, not provider output"},
		Unknowns: []models.Unknown{{Path: "deal.status", Reason: "DATA UJI unknown", ReferenceIDs: []string{"TEST-EVIDENCE"}}},
	}
}

func contextTestBounds() models.BoundInfo {
	cursor := "DATA-UJI-NEXT"
	return models.BoundInfo{Truncated: true, TruncationReasons: []string{"DATA UJI bound"}, NextCursor: &cursor}
}

func contextTestFact[T any](value *T, state string) models.Fact[T] {
	return models.Fact[T]{Value: value, State: state, TemporalBasis: "snapshot", EvidenceIDs: []string{"TEST-EVIDENCE"}, Limitations: []string{}}
}

func contextTestDeal() models.DealFacts {
	zero, zeroACV := 0, int64(0)
	return models.DealFacts{
		DealID: "TEST-DEAL", AccountID: "TEST-ACCOUNT",
		DealType: contextTestFact[string](nil, "snapshot_only"), OwnerID: contextTestFact[string](nil, "unknown"),
		Stage: contextTestFact[string](nil, "snapshot_only"), StageSince: contextTestFact[models.Date](nil, "snapshot_only"),
		CreatedAt: contextTestFact[models.Date](nil, "unknown"), PlannedOutlets: contextTestFact(&zero, "known"),
		PotentialACVIDR: contextTestFact(&zeroACV, "known"), Status: contextTestFact[string](nil, "ambiguous"),
		RecordEvidenceIDs: []string{"TEST-EVIDENCE"},
	}
}

func (s *contextHTTPStub) List(ctx context.Context, snapshot models.SnapshotContext, options models.DealListOptions) (models.DealFactsPage, error) {
	s.listOptions = options
	err := s.record(ctx, "list", "", snapshot)
	return models.DealFactsPage{Meta: contextTestMeta(snapshot), Bounds: contextTestBounds(), Items: []models.DealFacts{contextTestDeal()}}, err
}

func (s *contextHTTPStub) Deal(ctx context.Context, id string, snapshot models.SnapshotContext) (models.DealFactsResult, error) {
	err := s.record(ctx, "deal", id, snapshot)
	return models.DealFactsResult{Meta: contextTestMeta(snapshot), Deal: contextTestDeal()}, err
}

func (s *contextHTTPStub) Graph(ctx context.Context, id string, snapshot models.SnapshotContext, options models.GraphOptions) (models.GraphResult, error) {
	s.graphOptions = options
	err := s.record(ctx, "graph", id, snapshot)
	label := "DATA UJI node"
	return models.GraphResult{
		Meta: contextTestMeta(snapshot), Bounds: contextTestBounds(),
		Nodes: []models.Node{{NodeID: "TEST-NODE", NodeType: "deal", EntityID: "TEST-DEAL", Label: contextTestFact(&label, "known"),
			Validity: models.Validity{TemporalBasis: "undated"}, VerificationState: "verified", EvidenceIDs: []string{"TEST-EVIDENCE"}, EventIDs: []string{"TEST-EVENT"}}},
		Edges: []models.Edge{{EdgeID: "TEST-EDGE", EdgeType: "DATA-UJI", Source: "TEST-NODE", Target: "TEST-NODE",
			Validity: models.Validity{TemporalBasis: "undated"}, VerificationState: "verified", MatchMethod: "DATA-UJI",
			EvidenceIDs: []string{"TEST-EVIDENCE"}, EventIDs: []string{"TEST-EVENT"}}},
	}, err
}

func (s *contextHTTPStub) Timeline(ctx context.Context, id string, snapshot models.SnapshotContext, options models.TimelineOptions) (models.TimelineResult, error) {
	s.timeOptions = options
	err := s.record(ctx, "timeline", id, snapshot)
	summary := "DATA UJI event"
	return models.TimelineResult{Meta: contextTestMeta(snapshot), Bounds: contextTestBounds(), Events: []models.Event{{
		EventID: "TEST-EVENT", VerificationState: "verified", EventAt: snapshot.AsOf, TimePrecision: "date", EventType: "DATA-UJI",
		Status: contextTestFact[string](nil, "unknown"), Summary: contextTestFact(&summary, "known"),
		Actors: []models.ParticipantRef{}, Targets: []models.ParticipantRef{}, Participants: []models.ParticipantRef{},
		ScopeKind: "deal", DealIDs: []string{"TEST-DEAL"}, AccountIDs: []string{"TEST-ACCOUNT"},
		NodeIDs: []string{"TEST-NODE"}, EdgeIDs: []string{"TEST-EDGE"}, EvidenceIDs: []string{"TEST-EVIDENCE"},
	}}}, err
}

func (s *contextHTTPStub) Evidence(ctx context.Context, id string, snapshot models.SnapshotContext) (models.EvidenceResult, error) {
	err := s.record(ctx, "evidence", id, snapshot)
	return models.EvidenceResult{Meta: contextTestMeta(snapshot), Evidence: models.Evidence{
		EvidenceID: id, SourceFile: "DATA-UJI.json", SourceRecordID: "TEST-RECORD", RecordIDKind: "native",
		SourceKey: map[string]string{"deal_id": "TEST-DEAL"}, SourceChecksum: "DATA-UJI-CHECKSUM", SourceField: "DATA-UJI",
		SourceDate: &snapshot.AsOf, TemporalBasis: "event", ContentExcerpt: "DATA UJI, not business evidence", ScopeKind: "deal",
		VerificationState: "verified", DealIDs: []string{"TEST-DEAL"}, AccountIDs: []string{"TEST-ACCOUNT"},
		EventIDs: []string{"TEST-EVENT"}, EdgeIDs: []string{"TEST-EDGE"}, OccurrenceIDs: []string{},
	}}, err
}

type contextHTTPEndpoint struct {
	name, path, method, body string
	call                     func(*ContextController, http.ResponseWriter, *http.Request)
}

func contextReadEndpoints() []contextHTTPEndpoint {
	return []contextHTTPEndpoint{
		{"list", "/api/deals", http.MethodGet, "", (*ContextController).List},
		{"deal", "/api/deals/TEST-DEAL", http.MethodGet, "", (*ContextController).Get},
		{"graph", "/api/deals/TEST-DEAL/graph", http.MethodGet, "", (*ContextController).Graph},
		{"timeline", "/api/deals/TEST-DEAL/timeline", http.MethodGet, "", (*ContextController).Timeline},
		{"evidence", "/api/evidence/TEST-EVIDENCE", http.MethodGet, "", (*ContextController).Evidence},
	}
}

func contextOptionalEndpoints() []contextHTTPEndpoint {
	return []contextHTTPEndpoint{
		{"assessment", "/api/deals/TEST-DEAL/assessment", http.MethodGet, "", (*ContextController).Assessment},
		{"action_candidates", "/api/deals/TEST-DEAL/action-candidates", http.MethodGet, "", (*ContextController).ActionCandidates},
		{"comparison", "/api/deals/TEST-DEAL/actions/compare", http.MethodPost, `{"action_ids":["TEST-A","TEST-B"]}`, (*ContextController).Compare},
		{"copilot", "/api/copilot/ask", http.MethodPost, `{"deal_id":"TEST-DEAL","question":"DATA UJI question"}`, (*ContextController).Ask},
	}
}

func contextTestRequest(endpoint contextHTTPEndpoint, query string) *http.Request {
	r := httptest.NewRequest(endpoint.method, endpoint.path+query, strings.NewReader(endpoint.body))
	r.SetPathValue("deal_id", "TEST-DEAL")
	r.SetPathValue("evidence_id", "TEST-EVIDENCE")
	if endpoint.method == http.MethodPost {
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

func assertContextHTTPError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status || rec.Header().Get("Content-Type") != "application/json" || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("response status/headers = %d/%v, body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var body models.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Code != code || body.Message == "" {
		t.Fatalf("invalid error envelope: %s (decode %v)", rec.Body.String(), err)
	}
	if strings.Contains(rec.Body.String(), "SECRET") || strings.Contains(rec.Body.String(), "TEST-DEAL") {
		t.Fatalf("error leaked internal/entity data: %s", rec.Body.String())
	}
}

func TestContextControllerCanonicalEnvelopes(t *testing.T) {
	type requestKey struct{}
	for _, endpoint := range contextReadEndpoints() {
		for _, date := range []models.Date{repository.MaxAsOf, "2024-02-29"} {
			t.Run(endpoint.name+"/"+string(date), func(t *testing.T) {
				stub := &contextHTTPStub{onRead: func(ctx context.Context) error {
					deadline, ok := ctx.Deadline()
					if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > contextQueryTimeout || ctx.Value(requestKey{}) != "DATA-UJI" {
						t.Fatalf("request context/deadline not propagated: %v", ctx)
					}
					return nil
				}}
				query := ""
				if date != repository.MaxAsOf {
					query = "?as_of=" + string(date)
				}
				r := contextTestRequest(endpoint, query).WithContext(context.WithValue(context.Background(), requestKey{}, "DATA-UJI"))
				rec := httptest.NewRecorder()
				endpoint.call(NewContextController(stub), rec, r)
				if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json" || rec.Header().Get("Cache-Control") != "no-store" {
					t.Fatalf("unexpected response: %d, %v, %s", rec.Code, rec.Header(), rec.Body.String())
				}
				if stub.snapshotCalls != 1 || !reflect.DeepEqual(stub.reads, []string{endpoint.name}) || stub.asOf != date || stub.snapshot.AsOf != date {
					t.Fatalf("snapshot/read propagation failed: %+v", stub)
				}
				if err := repository.ValidateSnapshot(stub.snapshot); err != nil || stub.ctx.Err() != context.Canceled || r.Context().Err() != nil {
					t.Fatalf("snapshot consistency or timeout cleanup failed: snapshot=%v, query=%v, request=%v", err, stub.ctx.Err(), r.Context().Err())
				}
				wantID := "TEST-DEAL"
				if endpoint.name == "list" {
					wantID = ""
				} else if endpoint.name == "evidence" {
					wantID = "TEST-EVIDENCE"
				}
				if stub.id != wantID {
					t.Fatalf("path ID = %q, want %q", stub.id, wantID)
				}
				var body map[string]json.RawMessage
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				var meta models.Meta
				if err := json.Unmarshal(body["meta"], &meta); err != nil || !reflect.DeepEqual(meta, contextTestMeta(stub.snapshot)) {
					t.Fatalf("metadata lost: %+v (%v)", meta, err)
				}
				if strings.Contains(rec.Body.String(), "allowed_deal_ids") || strings.Contains(rec.Body.String(), `"access"`) || strings.Contains(rec.Body.String(), `"scope"`) {
					t.Fatalf("wire envelope leaked/replaced scope: %s", rec.Body.String())
				}
				if endpoint.name == "list" || endpoint.name == "graph" || endpoint.name == "timeline" {
					var bounds models.BoundInfo
					if err := json.Unmarshal(body["bounds"], &bounds); err != nil || !reflect.DeepEqual(bounds, contextTestBounds()) {
						t.Fatalf("bounds lost: %+v (%v)", bounds, err)
					}
				}
				switch endpoint.name {
				case "list", "deal":
					var deal models.DealFacts
					if endpoint.name == "list" {
						var items []models.DealFacts
						if err := json.Unmarshal(body["items"], &items); err != nil || len(items) != 1 || len(body) != 3 {
							t.Fatalf("not a canonical facts page: %s", rec.Body.String())
						}
						deal = items[0]
						if stub.listOptions.Limit != 50 || stub.listOptions.Cursor != nil {
							t.Fatalf("invalid list defaults: %+v", stub.listOptions)
						}
					} else if err := json.Unmarshal(body["deal"], &deal); err != nil || len(body) != 2 {
						t.Fatalf("not a canonical facts detail: %s", rec.Body.String())
					}
					if !reflect.DeepEqual(deal, contextTestDeal()) || deal.PlannedOutlets.Value == nil || *deal.PlannedOutlets.Value != 0 || deal.PotentialACVIDR.Value == nil || *deal.PotentialACVIDR.Value != 0 || deal.Status.Value != nil || deal.Stage.Value != nil {
						t.Fatalf("zero/null/state/provenance changed: %+v", deal)
					}
				case "graph":
					var graph models.GraphResult
					if err := json.Unmarshal(rec.Body.Bytes(), &graph); err != nil || len(body) != 4 || len(graph.Nodes) != 1 || len(graph.Edges) != 1 || graph.Edges[0].SourceTimestamp != nil || graph.Nodes[0].Label.EvidenceIDs[0] != "TEST-EVIDENCE" {
						t.Fatalf("graph provenance/nulls lost: %s", rec.Body.String())
					}
					if stub.graphOptions != (models.GraphOptions{Depth: 1, MaxNodes: 150, MaxEdges: 300}) {
						t.Fatalf("invalid graph defaults: %+v", stub.graphOptions)
					}
				case "timeline":
					var timeline models.TimelineResult
					if err := json.Unmarshal(rec.Body.Bytes(), &timeline); err != nil || len(body) != 3 || len(timeline.Events) != 1 || timeline.Events[0].EventAt != date || timeline.Events[0].SourceTimestamp != nil || timeline.Events[0].EvidenceIDs[0] != "TEST-EVIDENCE" {
						t.Fatalf("timeline provenance/date/nulls lost: %s", rec.Body.String())
					}
					if stub.timeOptions.Limit != 50 || stub.timeOptions.Cursor != nil {
						t.Fatalf("invalid timeline defaults: %+v", stub.timeOptions)
					}
				case "evidence":
					var evidence models.EvidenceResult
					if err := json.Unmarshal(rec.Body.Bytes(), &evidence); err != nil || len(body) != 2 || evidence.Evidence.SourceFile != "DATA-UJI.json" || evidence.Evidence.SourceChecksum != "DATA-UJI-CHECKSUM" || evidence.Evidence.SourceRecordID != "TEST-RECORD" || evidence.Evidence.SourceKey["deal_id"] != "TEST-DEAL" || evidence.Evidence.SourceTimestamp != nil || evidence.Evidence.Span != nil {
						t.Fatalf("evidence locator/provenance/nulls lost: %s", rec.Body.String())
					}
				}
			})
		}
	}
}

func TestContextControllerOptions(t *testing.T) {
	cursor, focus := "DATA-UJI-CURSOR", "TEST-EVENT"
	cases := []struct {
		index int
		query string
	}{
		{0, "?limit=7&cursor=" + cursor + "&account_ids=TEST-ACCOUNT&deal_types=DATA-UJI"},
		{2, "?depth=2&max_nodes=8&max_edges=12&focus_event_id=" + focus},
		{3, "?limit=3&cursor=" + cursor + "&event_ids=TEST-EVENT&event_types=DATA-UJI&actor_node_ids=TEST-ACTOR&statuses=DATA-UJI"},
	}
	for _, tc := range cases {
		endpoint := contextReadEndpoints()[tc.index]
		t.Run(endpoint.name, func(t *testing.T) {
			stub, rec := &contextHTTPStub{}, httptest.NewRecorder()
			endpoint.call(NewContextController(stub), rec, contextTestRequest(endpoint, tc.query))
			if rec.Code != http.StatusOK {
				t.Fatalf("valid options rejected: %s", rec.Body.String())
			}
			var got, want any
			switch endpoint.name {
			case "list":
				got, want = stub.listOptions, models.DealListOptions{Limit: 7, Cursor: &cursor, AccountIDs: []string{"TEST-ACCOUNT"}, DealTypes: []string{"DATA-UJI"}}
			case "graph":
				got, want = stub.graphOptions, models.GraphOptions{Depth: 2, MaxNodes: 8, MaxEdges: 12, FocusEventID: &focus}
			case "timeline":
				got, want = stub.timeOptions, models.TimelineOptions{Limit: 3, Cursor: &cursor, EventIDs: []string{"TEST-EVENT"}, EventTypes: []string{"DATA-UJI"}, ActorNodeIDs: []string{"TEST-ACTOR"}, Statuses: []string{"DATA-UJI"}}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("options = %+v, want %+v", got, want)
			}
		})
	}
}

func TestContextControllerStrictValidationBeforeReader(t *testing.T) {
	queries := []string{
		"?as_of=", "?as_of=invalid", "?as_of=2026-02-30", "?as_of=2025-02-29", "?as_of=0000-01-01", "?as_of=2026-10-02",
		"?as_of=2026-10-01T00:00:00Z", "?as_of=2026-2-01", "?as_of=%202026-10-01", "?as_of=2026-10-01&as_of=2026-10-01",
		"?as_of=%ZZ", "?ignored=%ZZ", "?as_of=2026-10-01;ignored=x",
		"?access=TEST", "?scope=TEST", "?allowed_deal_ids=TEST-DEAL", "?context_id=TEST", "?allowed_account_ids=TEST-ACCOUNT", "?allow_company_evidence=true",
	}
	for _, endpoint := range append(contextReadEndpoints(), contextOptionalEndpoints()...) {
		for i, query := range queries {
			t.Run(fmt.Sprintf("%s/%d", endpoint.name, i), func(t *testing.T) {
				for _, reader := range []ContextReader{nil, &contextHTTPStub{snapshotErr: &repository.RepositoryError{Code: repository.DataNotReady}}} {
					rec := httptest.NewRecorder()
					endpoint.call(NewContextController(reader), rec, contextTestRequest(endpoint, query))
					if rec.Code != http.StatusBadRequest || rec.Header().Get("Content-Type") != "application/json" || !json.Valid(rec.Body.Bytes()) {
						t.Fatalf("validation did not precede unavailable adapter: %d %s", rec.Code, rec.Body.String())
					}
					if stub, ok := reader.(*contextHTTPStub); ok && (stub.snapshotCalls != 0 || len(stub.reads) != 0) {
						t.Fatalf("invalid input reached reader: %+v", stub)
					}
				}
			})
		}
	}
}

func TestContextControllerInvalidOptions(t *testing.T) {
	for _, endpoint := range contextReadEndpoints() {
		fields := map[string][]string{}
		switch endpoint.name {
		case "list":
			fields = map[string][]string{"limit": {"", "0", "-1", "51", "1.5", "NaN", "+1", "99999999999999999999"}, "cursor": {"", " ", strings.Repeat("x", contextCursorLimit+1)}, "account_ids": {"", " ", strings.Repeat("x", contextIDLimit+1)}, "deal_types": {"", " "}}
		case "graph":
			fields = map[string][]string{"depth": {"", "0", "-1", "3", "1.5"}, "max_nodes": {"", "0", "151"}, "max_edges": {"", "0", "301"}, "focus_event_id": {"", " ", " TEST-EVENT", "TEST/EVENT", strings.Repeat("x", contextIDLimit+1)}}
		case "timeline":
			fields = map[string][]string{"limit": {"", "0", "51"}, "cursor": {"", " ", strings.Repeat("x", contextCursorLimit+1)}, "event_ids": {"", " "}, "event_types": {"", " "}, "actor_node_ids": {"", " "}, "statuses": {"", " "}}
		}
		for field, values := range fields {
			queries := []string{"?" + field + "=1&" + field + "=1"}
			for _, value := range values {
				queries = append(queries, "?"+field+"="+url.QueryEscape(value))
			}
			for i, query := range queries {
				t.Run(fmt.Sprintf("%s/%s/%d", endpoint.name, field, i), func(t *testing.T) {
					stub, rec := &contextHTTPStub{}, httptest.NewRecorder()
					endpoint.call(NewContextController(stub), rec, contextTestRequest(endpoint, query))
					assertContextHTTPError(t, rec, http.StatusBadRequest, "invalid_query")
					if stub.snapshotCalls != 0 || len(stub.reads) != 0 {
						t.Fatal("invalid/duplicate option reached reader")
					}
				})
			}
		}
	}
}

func TestContextControllerInvalidIDsAndMethods(t *testing.T) {
	for _, endpoint := range append(contextReadEndpoints(), contextOptionalEndpoints()...) {
		t.Run(endpoint.name, func(t *testing.T) {
			for _, method := range []string{http.MethodHead, http.MethodPut, http.MethodOptions} {
				stub, rec := &contextHTTPStub{}, httptest.NewRecorder()
				r := contextTestRequest(endpoint, "")
				r.Method = method
				endpoint.call(NewContextController(stub), rec, r)
				assertContextHTTPError(t, rec, http.StatusMethodNotAllowed, "method_not_allowed")
				if rec.Header().Get("Allow") != endpoint.method || stub.snapshotCalls != 0 {
					t.Fatal("incorrect method handling")
				}
			}
			if endpoint.name == "list" || endpoint.name == "copilot" {
				return
			}
			for _, id := range []string{"", " ", " TEST-DEAL", "TEST DEAL", "TEST/DEAL", "TEST\\DEAL", "TEST\nDEAL", strings.Repeat("x", contextIDLimit+1)} {
				stub, rec := &contextHTTPStub{}, httptest.NewRecorder()
				r := contextTestRequest(endpoint, "")
				key := "deal_id"
				if endpoint.name == "evidence" {
					key = "evidence_id"
				}
				r.SetPathValue(key, id)
				endpoint.call(NewContextController(stub), rec, r)
				assertContextHTTPError(t, rec, http.StatusBadRequest, "invalid_query")
				if stub.snapshotCalls != 0 {
					t.Fatal("invalid ID reached reader")
				}
			}
		})
	}
}

func TestContextControllerSafeReaderErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		code   string
		status int
	}{
		{"not found", &repository.RepositoryError{Code: repository.NotFound, EntityID: "SECRET"}, "not_found", 404},
		{"denied", &repository.RepositoryError{Code: repository.AccessDenied, EntityID: "SECRET"}, "not_found", 404},
		{"historically hidden", &repository.RepositoryError{Code: repository.NotVisible}, "not_visible_at_snapshot", 404},
		{"unavailable", &repository.RepositoryError{Code: repository.DataNotReady}, "data_not_ready", 503},
		{"cursor context", &repository.RepositoryError{Code: repository.ContextMismatch}, "context_mismatch", 409},
		{"query", &repository.RepositoryError{Code: repository.QueryFailed, Cause: errors.New("SECRET query")}, "query_failed", 500},
		{"unexpected", errors.New("SECRET driver"), "internal_error", 500},
		{"deadline", &repository.RepositoryError{Code: repository.QueryFailed, Cause: context.DeadlineExceeded}, "deadline_exceeded", 504},
		{"canceled", &repository.RepositoryError{Code: repository.QueryFailed, Cause: context.Canceled}, "", 0},
	}
	for _, endpoint := range append(contextReadEndpoints(), contextOptionalEndpoints()...) {
		for _, tc := range cases {
			t.Run(endpoint.name+"/"+tc.name, func(t *testing.T) {
				stub := &contextHTTPStub{readErr: fmt.Errorf("SECRET wrapper: %w", tc.err)}
				rec := httptest.NewRecorder()
				endpoint.call(NewContextController(stub), rec, contextTestRequest(endpoint, ""))
				if tc.status == 0 {
					if rec.Body.Len() != 0 || rec.Header().Get("Content-Type") != "" {
						t.Fatal("cancellation wrote a response")
					}
				} else {
					assertContextHTTPError(t, rec, tc.status, tc.code)
				}
				if stub.snapshotCalls != 1 || len(stub.reads) != 1 {
					t.Fatalf("unexpected reader calls: %+v", stub)
				}
			})
		}
	}
	for _, endpoint := range contextReadEndpoints() {
		rec := httptest.NewRecorder()
		endpoint.call(NewContextController(nil), rec, contextTestRequest(endpoint, ""))
		assertContextHTTPError(t, rec, http.StatusServiceUnavailable, "data_not_ready")
		stub := &contextHTTPStub{snapshotErr: &repository.RepositoryError{Code: repository.DataNotReady}}
		rec = httptest.NewRecorder()
		endpoint.call(NewContextController(stub), rec, contextTestRequest(endpoint, ""))
		assertContextHTTPError(t, rec, http.StatusServiceUnavailable, "data_not_ready")
		if len(stub.reads) != 0 {
			t.Fatal("snapshot failure reached query")
		}
	}
}

func TestContextControllerRequestCancellationAndDeadline(t *testing.T) {
	endpoint := contextReadEndpoints()[0]
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprint(deadline), func(t *testing.T) {
			var ctx context.Context
			var cancel context.CancelFunc
			if deadline {
				ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			} else {
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			}
			defer cancel()
			stub, rec := &contextHTTPStub{}, httptest.NewRecorder()
			endpoint.call(NewContextController(stub), rec, contextTestRequest(endpoint, "").WithContext(ctx))
			if deadline {
				assertContextHTTPError(t, rec, 504, "deadline_exceeded")
			} else if rec.Body.Len() != 0 {
				t.Fatal("canceled request wrote a response")
			}
			if stub.snapshotCalls != 0 {
				t.Fatal("finished request reached reader")
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	stub := &contextHTTPStub{onRead: func(queryCtx context.Context) error {
		deadline, _ := queryCtx.Deadline()
		want, _ := ctx.Deadline()
		if !deadline.Equal(want) {
			t.Fatal("query extended caller deadline")
		}
		<-queryCtx.Done()
		return nil // A reader ignoring cancellation must not produce a false 200.
	}}
	rec := httptest.NewRecorder()
	endpoint.call(NewContextController(stub), rec, contextTestRequest(endpoint, "").WithContext(ctx))
	assertContextHTTPError(t, rec, 504, "deadline_exceeded")
}

// DATA UJI assessor capability, separate from the unsupported reader stub.
type contextAssessmentStub struct {
	*contextHTTPStub
	assessment models.DealAssessment
}

func (s *contextAssessmentStub) Assessment(ctx context.Context, id string, snapshot models.SnapshotContext) (models.DealAssessment, error) {
	err := s.record(ctx, "assessment", id, snapshot)
	result := s.assessment
	result.Meta = contextTestMeta(snapshot)
	return result, err
}

func TestContextControllerAssessmentForwarding(t *testing.T) {
	type requestKey struct{}
	zero, zeroFloat := 0, float64(0)
	for _, status := range []string{"jev_unavailable", "insufficient_evidence", "complete"} {
		for _, query := range []string{"", "?as_of=2024-02-29"} {
			t.Run(status+query, func(t *testing.T) {
				assessment := models.DealAssessment{
					Status: status, RubricVersion: "DATA-UJI-RUBRIC", Explanation: "DATA UJI, not provider output",
					Unknowns: []string{"DATA UJI limitation"}, EvidenceIDs: []string{"TEST-EVIDENCE"},
				}
				if status == "complete" {
					assessment.Readiness100, assessment.JEVRawScore, assessment.JEVConfidence = &zero, &zeroFloat, &zeroFloat
				}
				stub := &contextAssessmentStub{contextHTTPStub: &contextHTTPStub{onRead: func(ctx context.Context) error {
					deadline, ok := ctx.Deadline()
					if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > 15*time.Second || ctx.Value(requestKey{}) != "DATA-UJI" {
						t.Fatalf("assessor context/deadline not forwarded: %v", ctx)
					}
					return nil
				}}, assessment: assessment}
				r := contextTestRequest(contextOptionalEndpoints()[0], query).WithContext(context.WithValue(context.Background(), requestKey{}, "DATA-UJI"))
				rec := httptest.NewRecorder()
				NewContextController(stub).Assessment(rec, r)
				if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json" || rec.Header().Get("Cache-Control") != "no-store" {
					t.Fatalf("canonical assessment not returned: %d %v %s", rec.Code, rec.Header(), rec.Body.String())
				}
				wantDate := repository.MaxAsOf
				if query != "" {
					wantDate = "2024-02-29"
				}
				if stub.snapshotCalls != 1 || !reflect.DeepEqual(stub.reads, []string{"assessment"}) || stub.id != "TEST-DEAL" || stub.asOf != wantDate || stub.snapshot.AsOf != wantDate {
					t.Fatalf("assessment used wrong snapshot/ID or fallback lookup: %+v", stub.contextHTTPStub)
				}
				if err := repository.ValidateSnapshot(stub.snapshot); err != nil || stub.ctx.Err() != context.Canceled || r.Context().Err() != nil {
					t.Fatalf("snapshot/timeout cleanup failed: %v, %v, %v", err, stub.ctx.Err(), r.Context().Err())
				}
				assessment.Meta = contextTestMeta(stub.snapshot)
				wantJSON, err := json.Marshal(assessment)
				if err != nil || rec.Body.String() != string(wantJSON)+"\n" {
					t.Fatalf("assessment fields/meta changed: got %s, want %s (%v)", rec.Body.String(), wantJSON, err)
				}
				var body map[string]json.RawMessage
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				for _, field := range []string{"readiness_100", "jev_raw_score", "jev_confidence"} {
					want := "null"
					if status == "complete" {
						want = "0"
					}
					if string(body[field]) != want {
						t.Fatalf("%s = %s, want %s", field, body[field], want)
					}
				}
			})
		}
	}
}

func TestContextControllerAssessmentErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"denied", &repository.RepositoryError{Code: repository.AccessDenied, EntityID: "SECRET"}, 404, "not_found"},
		{"not found", &repository.RepositoryError{Code: repository.NotFound}, 404, "not_found"},
		{"not visible", &repository.RepositoryError{Code: repository.NotVisible}, 404, "not_visible_at_snapshot"},
		{"adapter unavailable", &repository.RepositoryError{Code: repository.DataNotReady}, 503, "data_not_ready"},
		{"unexpected", errors.New("SECRET provider detail"), 500, "internal_error"},
		{"deadline", &repository.RepositoryError{Code: repository.QueryFailed, Cause: context.DeadlineExceeded}, 504, "deadline_exceeded"},
		{"canceled", &repository.RepositoryError{Code: repository.QueryFailed, Cause: context.Canceled}, 0, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &contextAssessmentStub{
				contextHTTPStub: &contextHTTPStub{readErr: fmt.Errorf("SECRET wrapper: %w", tc.err)},
				assessment:      models.DealAssessment{Status: "DATA-UJI-SHOULD-NOT-LEAK"},
			}
			rec := httptest.NewRecorder()
			NewContextController(stub).Assessment(rec, contextTestRequest(contextOptionalEndpoints()[0], ""))
			if tc.status == 0 {
				if rec.Body.Len() != 0 || rec.Header().Get("Content-Type") != "" {
					t.Fatal("canceled assessment wrote a response")
				}
			} else {
				assertContextHTTPError(t, rec, tc.status, tc.code)
				if strings.Contains(rec.Body.String(), "SHOULD-NOT-LEAK") {
					t.Fatal("assessment error published a result")
				}
			}
			if stub.snapshotCalls != 1 || !reflect.DeepEqual(stub.reads, []string{"assessment"}) {
				t.Fatalf("assessment error used fallback or bypassed reader: %+v", stub.contextHTTPStub)
			}
		})
	}
}

func TestContextControllerAssessmentValidatesBeforeAssessor(t *testing.T) {
	for _, query := range []string{"?as_of=", "?as_of=2026-02-30", "?as_of=2026-10-02", "?as_of=2026-10-01&as_of=2026-10-01", "?as_of=%ZZ", "?scope=TEST"} {
		stub := &contextAssessmentStub{contextHTTPStub: &contextHTTPStub{}}
		rec := httptest.NewRecorder()
		NewContextController(stub).Assessment(rec, contextTestRequest(contextOptionalEndpoints()[0], query))
		if rec.Code != http.StatusBadRequest || !json.Valid(rec.Body.Bytes()) || stub.snapshotCalls != 0 || len(stub.reads) != 0 {
			t.Fatalf("invalid query reached assessor: %d %s, %+v", rec.Code, rec.Body.String(), stub.contextHTTPStub)
		}
	}
}

func TestContextControllerOptionalUnavailable(t *testing.T) {
	for _, endpoint := range contextOptionalEndpoints() {
		for _, query := range []string{"", "?as_of=2024-02-29"} {
			t.Run(endpoint.name+query, func(t *testing.T) {
				stub, rec := &contextHTTPStub{}, httptest.NewRecorder()
				endpoint.call(NewContextController(stub), rec, contextTestRequest(endpoint, query))
				assertContextHTTPError(t, rec, 503, endpoint.name+"_unavailable")
				wantDate := repository.MaxAsOf
				if query != "" {
					wantDate = "2024-02-29"
				}
				if stub.snapshotCalls != 1 || !reflect.DeepEqual(stub.reads, []string{"deal"}) || stub.id != "TEST-DEAL" || stub.snapshot.AsOf != wantDate {
					t.Fatalf("unavailable endpoint skipped scoped deal lookup: %+v", stub)
				}
				var body map[string]any
				_ = json.Unmarshal(rec.Body.Bytes(), &body)
				if len(body) != 2 {
					t.Fatalf("unavailable endpoint invented results: %s", rec.Body.String())
				}
			})
		}
	}
	for count := 2; count <= 4; count++ {
		endpoint := contextOptionalEndpoints()[2]
		ids := []string{"TEST-A", "TEST-B", "TEST-C", "TEST-D"}[:count]
		body, _ := json.Marshal(map[string]any{"as_of": "2024-02-29", "action_ids": ids})
		endpoint.body = string(body)
		stub, rec := &contextHTTPStub{}, httptest.NewRecorder()
		endpoint.call(NewContextController(stub), rec, contextTestRequest(endpoint, ""))
		assertContextHTTPError(t, rec, 503, "comparison_unavailable")
		if stub.asOf != "2024-02-29" {
			t.Fatal("body historical date not propagated")
		}
	}
	endpoint := contextOptionalEndpoints()[3]
	body, _ := json.Marshal(map[string]any{"as_of": "2024-02-29", "deal_id": "TEST-DEAL", "question": strings.Repeat("界", 2000)})
	endpoint.body = string(body)
	stub, rec := &contextHTTPStub{}, httptest.NewRecorder()
	r := contextTestRequest(endpoint, "")
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	endpoint.call(NewContextController(stub), rec, r)
	assertContextHTTPError(t, rec, 503, "copilot_unavailable")
	if stub.asOf != "2024-02-29" {
		t.Fatal("copilot body historical date not propagated")
	}
}

func TestContextControllerOptionalBodyValidation(t *testing.T) {
	for _, endpoint := range contextOptionalEndpoints()[2:] {
		bodies := []string{"", "null", "[]", `"DATA UJI"`, `{`, endpoint.body + `{}`, endpoint.body + ` trailing`, strings.TrimSuffix(endpoint.body, "}") + `,"scope":{}}`, strings.TrimSuffix(endpoint.body, "}") + `,"as_of":null}`}
		for _, rawDate := range []string{`""`, `"2026-10-02"`, `"2026-02-30"`, `"2026-10-01T00:00:00Z"`, `1`, `true`} {
			bodies = append(bodies, strings.TrimSuffix(endpoint.body, "}")+`,"as_of":`+rawDate+`}`)
		}
		bodies = append(bodies, strings.TrimSuffix(endpoint.body, "}")+`,"as_of":"2026-10-01","as_of":"2026-10-01"}`)
		if endpoint.name == "comparison" {
			for _, ids := range []string{`null`, `[]`, `["TEST-A"]`, `["TEST-A","TEST-A"]`, `["TEST-A","TEST-B","TEST-C","TEST-D","TEST-E"]`, `["TEST-A",1]`, `["TEST-A",null]`, `["TEST-A",""]`, `["TEST-A"," TEST-B"]`, `"TEST-A"`} {
				bodies = append(bodies, `{"action_ids":`+ids+`}`)
			}
			bodies = append(bodies, `{}`, `{"action_ids":["TEST-A","TEST-B"],"action_ids":["TEST-A","TEST-C"]}`)
		} else {
			for _, body := range []string{
				`{"question":"DATA UJI"}`, `{"deal_id":null,"question":"DATA UJI"}`, `{"deal_id":"","question":"DATA UJI"}`,
				`{"deal_id":"TEST-DEAL"}`, `{"deal_id":"TEST-DEAL","question":null}`, `{"deal_id":"TEST-DEAL","question":1}`,
				`{"deal_id":"TEST-DEAL","question":" "}`, `{"deal_id":"TEST-DEAL","question":"A","question":"B"}`,
			} {
				bodies = append(bodies, body)
			}
			bodies = append(bodies, `{"deal_id":"TEST-DEAL","question":"`+strings.Repeat("x", 2001)+`"}`)
		}
		for i, body := range bodies {
			t.Run(fmt.Sprintf("%s/%d", endpoint.name, i), func(t *testing.T) {
				endpoint.body = body
				stub, rec := &contextHTTPStub{}, httptest.NewRecorder()
				endpoint.call(NewContextController(stub), rec, contextTestRequest(endpoint, ""))
				if rec.Code != 400 || rec.Header().Get("Content-Type") != "application/json" || !json.Valid(rec.Body.Bytes()) || stub.snapshotCalls != 0 || len(stub.reads) != 0 {
					t.Fatalf("invalid body reached reader or wrong response: %d %s, %+v", rec.Code, rec.Body.String(), stub)
				}
			})
		}
		endpoint = contextOptionalEndpoints()[map[string]int{"comparison": 2, "copilot": 3}[endpoint.name]]
		for _, contentType := range []string{"", "text/plain", "application/json;bad"} {
			rec, stub := httptest.NewRecorder(), &contextHTTPStub{}
			r := contextTestRequest(endpoint, "")
			r.Header.Set("Content-Type", contentType)
			endpoint.call(NewContextController(stub), rec, r)
			assertContextHTTPError(t, rec, 415, "unsupported_media_type")
			if stub.snapshotCalls != 0 {
				t.Fatal("wrong content type reached reader")
			}
		}
		endpoint.body = strings.TrimSuffix(endpoint.body, "}") + `,"as_of":"2026-10-01"}`
		rec, stub := httptest.NewRecorder(), &contextHTTPStub{}
		endpoint.call(NewContextController(stub), rec, contextTestRequest(endpoint, "?as_of=2026-10-01"))
		assertContextHTTPError(t, rec, 400, "invalid_snapshot")
		if stub.snapshotCalls != 0 {
			t.Fatal("ambiguous query/body date reached reader")
		}
		endpoint.body += strings.Repeat(" ", contextBodyLimit)
		rec = httptest.NewRecorder()
		endpoint.call(NewContextController(stub), rec, contextTestRequest(endpoint, ""))
		assertContextHTTPError(t, rec, 413, "request_too_large")
		if stub.snapshotCalls != 0 {
			t.Fatal("oversized body reached reader")
		}
	}
}
