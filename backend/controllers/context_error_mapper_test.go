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

	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
)

func TestMapContextRepositoryError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "nil error",
			err:        nil,
			wantStatus: http.StatusOK,
			wantCode:   "",
		},
		{
			name: "not found",
			err: &repository.RepositoryError{
				Code:       repository.NotFound,
				EntityKind: "deal",
				EntityID:   "DL-SECRET-999",
				Cause:      errors.New("db record not found"),
			},
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
		},
		{
			name: "not visible at snapshot",
			err: &repository.RepositoryError{
				Code:       repository.NotVisible,
				EntityKind: "evidence",
				EntityID:   "EV-FUTURE-123",
			},
			wantStatus: http.StatusNotFound,
			wantCode:   "not_visible_at_snapshot",
		},
		{
			name: "data not ready",
			err: &repository.RepositoryError{
				Code: repository.DataNotReady,
			},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   "data_not_ready",
		},
		{
			name: "access denied",
			err: &repository.RepositoryError{
				Code: repository.AccessDenied,
			},
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
		},
		{
			name: "invalid snapshot",
			err: &repository.RepositoryError{
				Code: repository.InvalidSnapshot,
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_snapshot",
		},
		{
			name: "invalid query",
			err: &repository.RepositoryError{
				Code: repository.InvalidQuery,
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_query",
		},
		{
			name: "invalid selection",
			err: &repository.RepositoryError{
				Code: repository.InvalidSelection,
				Issues: []repository.LookupIssue{
					{ID: "ACT-001", Code: repository.NotFound},
				},
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "invalid_selection",
		},
		{
			name: "context mismatch",
			err: &repository.RepositoryError{
				Code: repository.ContextMismatch,
			},
			wantStatus: http.StatusConflict,
			wantCode:   "context_mismatch",
		},
		{
			name: "bundle limit exceeded",
			err: &repository.RepositoryError{
				Code: repository.BundleLimitExceeded,
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "bundle_limit_exceeded",
		},
		{
			name: "ambiguous reference",
			err: &repository.RepositoryError{
				Code:       repository.AmbiguousReference,
				EntityKind: "account",
				EntityID:   "ACC-CONFIDENTIAL",
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "ambiguous_reference",
		},
		{
			name: "query failed",
			err: &repository.RepositoryError{
				Code:  repository.QueryFailed,
				Cause: errors.New("FATAL: connection to db server refused at 10.0.0.5:5432"),
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "query_failed",
		},
		{
			name:       "unclassified generic error",
			err:        errors.New("unexpected disk I/O panic"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tc := range cases {
		errs := []error{tc.err}
		if tc.err != nil {
			errs = append(errs, fmt.Errorf("SECRET outer wrapper: %w", tc.err))
		}
		for i, err := range errs {
			t.Run(fmt.Sprintf("%s/wrap_%d", tc.name, i), func(t *testing.T) {
				status, resp := MapContextRepositoryError(err)
				if status != tc.wantStatus {
					t.Errorf("status = %d, want %d", status, tc.wantStatus)
				}
				if resp.Code != tc.wantCode {
					t.Errorf("code = %q, want %q", resp.Code, tc.wantCode)
				}
			})
		}
	}
}

func TestWriteContextErrorConcealsAccessDenied(t *testing.T) {
	notFound := httptest.NewRecorder()
	WriteContextError(notFound, &repository.RepositoryError{Code: repository.NotFound})

	denied := &repository.RepositoryError{
		Code: repository.AccessDenied, EntityKind: "secret_table", EntityID: "SECRET-ID",
		Issues: []repository.LookupIssue{{ID: "SECRET-selected", Code: repository.NotFound}},
		Cause:  errors.New("SECRET connection string"),
	}
	for i, err := range []error{denied, fmt.Errorf("SECRET wrapper: %w", denied)} {
		t.Run(fmt.Sprintf("wrap_%d", i), func(t *testing.T) {
			rec := httptest.NewRecorder()
			WriteContextError(rec, err)
			if rec.Code != http.StatusNotFound || rec.Body.String() != notFound.Body.String() ||
				rec.Header().Get("Content-Type") != notFound.Header().Get("Content-Type") {
				t.Fatalf("access denial differs from not found: status=%d, body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestMapContextRepositoryErrorContextCauses(t *testing.T) {
	cases := []struct {
		name       string
		cause      error
		wantStatus int
		wantBody   models.ErrorResponse
	}{
		{name: "canceled", cause: context.Canceled, wantStatus: 0},
		{
			name: "deadline", cause: context.DeadlineExceeded, wantStatus: http.StatusGatewayTimeout,
			wantBody: models.ErrorResponse{Code: "deadline_exceeded", Message: "context query deadline exceeded"},
		},
	}
	for _, tc := range cases {
		errs := []error{
			tc.cause,
			fmt.Errorf("SECRET cause wrapper: %w", tc.cause),
			&repository.RepositoryError{Code: repository.QueryFailed, Cause: tc.cause},
			fmt.Errorf("SECRET outer wrapper: %w", &repository.RepositoryError{
				Code: repository.AccessDenied, EntityID: "SECRET-ID",
				Cause: fmt.Errorf("SECRET inner wrapper: %w", tc.cause),
			}),
			errors.Join(errors.New("SECRET driver details"), tc.cause),
			contextCauseMatcher{tc.cause},
		}
		for i, err := range errs {
			t.Run(fmt.Sprintf("%s/wrap_%d", tc.name, i), func(t *testing.T) {
				status, body := MapContextRepositoryError(err)
				if status != tc.wantStatus || body != tc.wantBody {
					t.Errorf("mapped (%d, %+v), want (%d, %+v)", status, body, tc.wantStatus, tc.wantBody)
				}
				if tc.wantStatus == 0 {
					WriteContextError(untouchedContextErrorWriter{t}, err)
					return
				}
				rec := httptest.NewRecorder()
				WriteContextError(rec, err)
				var written models.ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &written); err != nil {
					t.Fatalf("decode deadline response: %v", err)
				}
				if rec.Code != tc.wantStatus || written != tc.wantBody || rec.Header().Get("Content-Type") != "application/json" {
					t.Fatalf("unexpected deadline response: status=%d, body=%s", rec.Code, rec.Body.String())
				}
			})
		}
	}
}

func TestMapContextRepositoryErrorTypedNil(t *testing.T) {
	var repoErr *repository.RepositoryError
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"direct", repoErr, http.StatusInternalServerError, "internal_error"},
		{"wrapped", fmt.Errorf("SECRET wrapper: %w", repoErr), http.StatusInternalServerError, "internal_error"},
		{"repository cause", &repository.RepositoryError{Code: repository.QueryFailed, Cause: repoErr}, http.StatusInternalServerError, "query_failed"},
		{"nested cause", fmt.Errorf("SECRET outer: %w", &repository.RepositoryError{
			Code: repository.QueryFailed, Cause: fmt.Errorf("SECRET inner: %w", repoErr),
		}), http.StatusInternalServerError, "query_failed"},
		{"joined", errors.Join(errors.New("SECRET details"), repoErr), http.StatusInternalServerError, "internal_error"},
		{"joined after repository", errors.Join(&repository.RepositoryError{Code: repository.QueryFailed}, repoErr), http.StatusInternalServerError, "query_failed"},
		{"joined deadline", errors.Join(repoErr, context.DeadlineExceeded), http.StatusGatewayTimeout, "deadline_exceeded"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					t.Fatalf("typed-nil RepositoryError panicked: %v", p)
				}
			}()
			status, body := MapContextRepositoryError(tc.err)
			if status != tc.wantStatus || body.Code != tc.wantCode || body.Message == "" {
				t.Errorf("mapped (%d, %+v), want (%d, %s)", status, body, tc.wantStatus, tc.wantCode)
			}
			rec := httptest.NewRecorder()
			WriteContextError(rec, tc.err)
			var written models.ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &written); err != nil {
				t.Fatalf("decode typed-nil response: %v", err)
			}
			if rec.Code != tc.wantStatus || written != body {
				t.Fatalf("unexpected typed-nil response: status=%d, body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestWriteContextErrorNoOp(t *testing.T) {
	var repoErr *repository.RepositoryError
	cases := []struct {
		name string
		err  error
	}{
		{"nil", nil},
		{"canceled", context.Canceled},
		{"wrapped", fmt.Errorf("SECRET wrapper: %w", context.Canceled)},
		{"repository cause", &repository.RepositoryError{Code: repository.QueryFailed, Cause: context.Canceled}},
		{"wrapped repository", fmt.Errorf("SECRET wrapper: %w", &repository.RepositoryError{Code: repository.QueryFailed, Cause: context.Canceled})},
		{"cancellation before deadline", errors.Join(context.Canceled, context.DeadlineExceeded)},
		{"cancellation after deadline", errors.Join(context.DeadlineExceeded, context.Canceled)},
		{"cancellation after typed nil", errors.Join(repoErr, context.Canceled)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err != nil {
				status, body := MapContextRepositoryError(tc.err)
				if status != 0 || body != (models.ErrorResponse{}) {
					t.Errorf("cancellation mapped (%d, %+v), want no-response sentinel", status, body)
				}
			}
			WriteContextError(untouchedContextErrorWriter{t}, tc.err)
		})
	}
}

type contextCauseMatcher struct{ cause error }

func (e contextCauseMatcher) Error() string        { return "SECRET driver details" }
func (e contextCauseMatcher) Is(target error) bool { return target == e.cause }

// Fail even on Header(), so a no-op cannot silently mutate response headers.
type untouchedContextErrorWriter struct{ t *testing.T }

func (w untouchedContextErrorWriter) Header() http.Header {
	w.t.Fatal("no-op error response accessed headers")
	return nil
}
func (w untouchedContextErrorWriter) WriteHeader(int) {
	w.t.Fatal("no-op error response wrote a status")
}
func (w untouchedContextErrorWriter) Write([]byte) (int, error) {
	w.t.Fatal("no-op error response wrote a body")
	return 0, nil
}

func TestWriteContextErrorRedactsInternalDetails(t *testing.T) {
	cases := []struct {
		code       repository.ErrorCode
		wantStatus int
		wantCode   string
	}{
		{repository.NotFound, http.StatusNotFound, "not_found"},
		{repository.NotVisible, http.StatusNotFound, "not_visible_at_snapshot"},
		{repository.DataNotReady, http.StatusServiceUnavailable, "data_not_ready"},
		{repository.AccessDenied, http.StatusNotFound, "not_found"},
		{repository.InvalidSnapshot, http.StatusBadRequest, "invalid_snapshot"},
		{repository.InvalidQuery, http.StatusBadRequest, "invalid_query"},
		{repository.InvalidSelection, http.StatusUnprocessableEntity, "invalid_selection"},
		{repository.ContextMismatch, http.StatusConflict, "context_mismatch"},
		{repository.BundleLimitExceeded, http.StatusUnprocessableEntity, "bundle_limit_exceeded"},
		{repository.AmbiguousReference, http.StatusUnprocessableEntity, "ambiguous_reference"},
		{repository.QueryFailed, http.StatusInternalServerError, "query_failed"},
		{repository.ErrorCode("SECRET unknown category"), http.StatusInternalServerError, "internal_error"},
	}
	for _, tc := range cases {
		sensitiveErr := &repository.RepositoryError{
			Code: tc.code, EntityKind: "secret_table", EntityID: "ACC-LEAK-9999",
			Issues: []repository.LookupIssue{{ID: "ACT-SECRET-ID", Code: repository.NotFound}},
			Cause:  errors.New("driver trace: password 'super_secret' was rejected by postgres://admin@db:5432"),
		}
		for i, err := range []error{sensitiveErr, fmt.Errorf("SECRET wrapper: %w", sensitiveErr)} {
			t.Run(fmt.Sprintf("%s/wrap_%d", tc.code, i), func(t *testing.T) {
				rec := httptest.NewRecorder()
				WriteContextError(rec, err)
				if rec.Code != tc.wantStatus || rec.Header().Get("Content-Type") != "application/json" {
					t.Fatalf("unexpected response: status=%d, headers=%v", rec.Code, rec.Header())
				}
				rawBody := rec.Body.String()
				for _, s := range []string{"secret_table", "ACC-LEAK-9999", "ACT-SECRET-ID", "super_secret", "postgres", "driver trace", "SECRET"} {
					if strings.Contains(rawBody, s) {
						t.Fatalf("response body leaked sensitive detail %q: %s", s, rawBody)
					}
				}
				var parsed models.ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
					t.Fatalf("unmarshal error response: %v", err)
				}
				if parsed.Code != tc.wantCode || parsed.Message == "" {
					t.Errorf("unexpected public error: %+v", parsed)
				}
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(rec.Body.Bytes(), &fields); err != nil || len(fields) != 2 {
					t.Fatalf("response must contain only code/message: %s", rawBody)
				}
			})
		}
	}
}
