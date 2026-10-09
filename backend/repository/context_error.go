package repository

type ErrorCode string

const (
	NotFound            ErrorCode = "not_found"
	NotVisible          ErrorCode = "not_visible_at_snapshot"
	DataNotReady        ErrorCode = "data_not_ready"
	AmbiguousReference  ErrorCode = "ambiguous_reference"
	QueryFailed         ErrorCode = "query_failed"
	InvalidSnapshot     ErrorCode = "invalid_snapshot"
	InvalidQuery        ErrorCode = "invalid_query"
	AccessDenied        ErrorCode = "access_denied"
	ContextMismatch     ErrorCode = "context_mismatch"
	InvalidSelection    ErrorCode = "invalid_selection"
	BundleLimitExceeded ErrorCode = "bundle_limit_exceeded"
)

// LookupIssue is internal. O2 must redact it before exposing batch details.
type LookupIssue struct {
	ID   string
	Code ErrorCode
}

// RepositoryError is not an HTTP response. Error and JSON contain only the
// category; causes and entity references are available to trusted server code.
type RepositoryError struct {
	Code       ErrorCode     `json:"code"`
	EntityKind string        `json:"-"`
	EntityID   string        `json:"-"`
	Issues     []LookupIssue `json:"-"`
	Cause      error         `json:"-"`
}

func (e *RepositoryError) Error() string { return string(e.Code) }
func (e *RepositoryError) Unwrap() error { return e.Cause }
