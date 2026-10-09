package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/nopaalh/Relio/backend/repository"
)

func TestRepositoryErrorPreservesCause(t *testing.T) {
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		err := &repository.RepositoryError{Code: repository.QueryFailed, Cause: cause}
		if !errors.Is(err, cause) {
			t.Fatalf("cause lost: %v", cause)
		}
	}
}

func TestRepositoryErrorSafeMessage(t *testing.T) {
	err := &repository.RepositoryError{
		Code: repository.QueryFailed, EntityKind: "SECRET-kind", EntityID: "SECRET-account",
		Issues: []repository.LookupIssue{{ID: "SECRET-selected", Code: repository.AccessDenied}},
		Cause:  errors.New("SECRET-password query driver exception"),
	}
	if err.Error() != "query_failed" {
		t.Fatalf("unsafe message: %s", err.Error())
	}
	encoded, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if strings.Contains(string(encoded), "SECRET") {
		t.Fatalf("internal details serialized: %s", encoded)
	}
	if string(encoded) != `{"code":"query_failed"}` {
		t.Fatalf("unexpected repository serialization: %s", encoded)
	}
}

func TestRepositoryErrorAsThroughWrap(t *testing.T) {
	original := &repository.RepositoryError{Code: repository.NotVisible}
	err := fmt.Errorf("operation: %w", original)
	var got *repository.RepositoryError
	if !errors.As(err, &got) || got != original {
		t.Fatal("typed category lost through wrapping")
	}
}

func TestRepositoryErrorDistinctCodes(t *testing.T) {
	codes := map[repository.ErrorCode]string{
		repository.NotFound: "not_found", repository.NotVisible: "not_visible_at_snapshot",
		repository.DataNotReady: "data_not_ready", repository.AmbiguousReference: "ambiguous_reference",
		repository.QueryFailed: "query_failed", repository.InvalidSnapshot: "invalid_snapshot",
		repository.InvalidQuery: "invalid_query", repository.AccessDenied: "access_denied",
		repository.ContextMismatch: "context_mismatch", repository.InvalidSelection: "invalid_selection",
		repository.BundleLimitExceeded: "bundle_limit_exceeded",
	}
	if len(codes) != 11 {
		t.Fatalf("error categories collapsed: %d", len(codes))
	}
	for code, want := range codes {
		if (&repository.RepositoryError{Code: code}).Error() != want {
			t.Fatalf("category %q != %q", code, want)
		}
	}
}
