package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
)

// MapContextRepositoryError maps a repository error to an HTTP status code and
// a safe models.ErrorResponse. Internal causes, database details, lookup issues,
// and raw entity IDs are strictly redacted and never exposed to the caller.
// This resource-lookup mapper conceals access denial as not found.
// Status 0 denotes cancellation; callers must not write a response.
func MapContextRepositoryError(err error) (int, models.ErrorResponse) {
	if err == nil {
		return http.StatusOK, models.ErrorResponse{}
	}

	if contextErrorIs(err, context.Canceled) {
		return 0, models.ErrorResponse{}
	}
	if contextErrorIs(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout, models.ErrorResponse{
			Code:    "deadline_exceeded",
			Message: "context query deadline exceeded",
		}
	}

	var repoErr *repository.RepositoryError
	if errors.As(err, &repoErr) && repoErr != nil {
		switch repoErr.Code {
		case repository.NotFound, repository.AccessDenied:
			return http.StatusNotFound, models.ErrorResponse{
				Code:    string(repository.NotFound),
				Message: "requested entity was not found",
			}
		case repository.NotVisible:
			return http.StatusNotFound, models.ErrorResponse{
				Code:    string(repository.NotVisible),
				Message: "requested entity is not visible at the requested snapshot",
			}
		case repository.DataNotReady:
			return http.StatusServiceUnavailable, models.ErrorResponse{
				Code:    string(repository.DataNotReady),
				Message: "context data adapter is not ready or still initializing",
			}

		case repository.InvalidSnapshot:
			return http.StatusBadRequest, models.ErrorResponse{
				Code:    string(repository.InvalidSnapshot),
				Message: "snapshot parameters are invalid or outside the allowed historical range",
			}
		case repository.InvalidQuery:
			return http.StatusBadRequest, models.ErrorResponse{
				Code:    string(repository.InvalidQuery),
				Message: "query bounds or filter parameters are invalid",
			}
		case repository.InvalidSelection:
			return http.StatusUnprocessableEntity, models.ErrorResponse{
				Code:    string(repository.InvalidSelection),
				Message: "selected actions contain invalid, duplicate, or unresolvable items",
			}
		case repository.ContextMismatch:
			return http.StatusConflict, models.ErrorResponse{
				Code:    string(repository.ContextMismatch),
				Message: "request context mismatch detected against snapshot state",
			}
		case repository.BundleLimitExceeded:
			return http.StatusUnprocessableEntity, models.ErrorResponse{
				Code:    string(repository.BundleLimitExceeded),
				Message: "requested evidence bundle exceeds the allowed capacity",
			}
		case repository.AmbiguousReference:
			// Redacted: do not expose conflicting IDs to clients.
			return http.StatusUnprocessableEntity, models.ErrorResponse{
				Code:    string(repository.AmbiguousReference),
				Message: "entity reference is ambiguous and cannot be resolved deterministically",
			}
		case repository.QueryFailed:
			// Redacted: do not expose database driver, query, or stack traces.
			return http.StatusInternalServerError, models.ErrorResponse{
				Code:    string(repository.QueryFailed),
				Message: "internal context query execution failed",
			}
		}
	}

	// Default fallback for unclassified errors.
	return http.StatusInternalServerError, models.ErrorResponse{
		Code:    "internal_error",
		Message: "an unexpected error occurred while processing the request",
	}
}

// contextErrorIs follows Is/Unwrap like errors.Is, but stops at typed-nil
// RepositoryErrors because O1's Unwrap method is not nil-safe.
func contextErrorIs(err, target error) bool {
	for err != nil {
		if repoErr, ok := err.(*repository.RepositoryError); ok && repoErr == nil {
			return false
		}
		if err == target {
			return true
		}
		if matcher, ok := err.(interface{ Is(error) bool }); ok && matcher.Is(target) {
			return true
		}
		switch wrapped := err.(type) {
		case interface{ Unwrap() error }:
			err = wrapped.Unwrap()
		case interface{ Unwrap() []error }:
			for _, cause := range wrapped.Unwrap() {
				if contextErrorIs(cause, target) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
	return false
}

// WriteContextError writes a safe JSON error response to w.
// Nil errors and cancellation do not access headers or write a response.
func WriteContextError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	status, body := MapContextRepositoryError(err)
	if status == 0 {
		return
	}
	writeJSON(w, status, body)
}
