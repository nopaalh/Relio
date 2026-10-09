package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nopaalh/Relio/backend/models"
)

const (
	// JEVDefaultEndpoint is the documented production evaluation endpoint.
	JEVDefaultEndpoint = "https://api.typesafe.ai/v1/systemone"
	// JEVDefaultTimeout bounds one call, including reading the body.
	JEVDefaultTimeout = 90 * time.Second
	// JEVDefaultMaxResponseBytes bounds the success body that is decoded.
	JEVDefaultMaxResponseBytes int64 = 1 << 20
	// jevMaxErrorDrainBytes bounds how much of an error body is discarded so
	// the connection can be reused. Error bodies are never surfaced.
	jevMaxErrorDrainBytes int64 = 64 << 10
)

// Error kinds. Use errors.Is(err, ErrJEV...) to classify a failure. Context
// cancellation/deadline errors remain matchable with errors.Is as well.
var (
	ErrJEVNotConfigured   = errors.New("jev: client not configured")
	ErrJEVInvalidRequest  = errors.New("jev: invalid request")
	ErrJEVAuthentication  = errors.New("jev: authentication failed")
	ErrJEVRateLimited     = errors.New("jev: rate limited")
	ErrJEVProviderFailure = errors.New("jev: provider failure")
	ErrJEVTransport       = errors.New("jev: transport failure")
	ErrJEVInvalidResponse = errors.New("jev: invalid response")
)

// JEVError carries a classified failure. Client-generated messages omit secrets,
// source data, provider bodies, and underlying cause text; causes remain unwrap-able.
type JEVError struct {
	Kind       error
	StatusCode int    // 0 when no HTTP response was received
	RetryAfter string // Retry-After header value on 429/529, if any
	Detail     string // safe, Relio-generated description
	Err        error  // underlying cause (e.g. context.DeadlineExceeded)
}

func (e *JEVError) Error() string {
	var b strings.Builder
	b.WriteString(e.Kind.Error())
	if e.StatusCode != 0 {
		fmt.Fprintf(&b, " (HTTP %d)", e.StatusCode)
	}
	if e.Detail != "" {
		b.WriteString(": ")
		b.WriteString(e.Detail)
	}
	return b.String()
}

func (e *JEVError) Unwrap() []error {
	if e.Err == nil {
		return []error{e.Kind}
	}
	return []error{e.Kind, e.Err}
}

// JEVClientConfig configures JEVClient. Only APIKey is required.
type JEVClientConfig struct {
	APIKey string
	// Endpoint defaults to JEVDefaultEndpoint. Non-HTTPS endpoints are only
	// accepted for loopback hosts (httptest), so the key is never sent in
	// clear text over a network.
	Endpoint string
	// Model is used when a request leaves Model empty. Defaults to jev-latest.
	Model string
	// HTTPClient defaults to a client with JEVDefaultTimeout. TLS settings of
	// an injected client are used as-is; this package never disables
	// verification. The client is copied and redirects are disabled.
	HTTPClient *http.Client
	// Timeout is applied per call via context in addition to any caller
	// deadline. Defaults to JEVDefaultTimeout.
	Timeout time.Duration
	// MaxResponseBytes defaults to JEVDefaultMaxResponseBytes.
	MaxResponseBytes int64
}

// JEVClient is a minimal adapter for the TypeSafe System One endpoint. It
// performs one HTTP request per Evaluate call: no redirects or application
// retries, so a failure never turns into a paid-call loop.
type JEVClient struct {
	apiKey   string
	endpoint string
	model    string
	http     *http.Client
	timeout  time.Duration
	maxBytes int64
}

func NewJEVClient(cfg JEVClientConfig) (*JEVClient, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, &JEVError{Kind: ErrJEVNotConfigured, Detail: "API key is empty"}
	}
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = JEVDefaultEndpoint
	}
	if err := checkJEVEndpoint(endpoint); err != nil {
		return nil, &JEVError{Kind: ErrJEVNotConfigured, Detail: err.Error()}
	}
	c := &JEVClient{
		apiKey:   cfg.APIKey,
		endpoint: endpoint,
		model:    cfg.Model,
		http:     cfg.HTTPClient,
		timeout:  cfg.Timeout,
		maxBytes: cfg.MaxResponseBytes,
	}
	if c.model == "" {
		c.model = models.JEVModelLatest
	}
	if c.timeout <= 0 {
		c.timeout = JEVDefaultTimeout
	}
	if c.http == nil {
		c.http = &http.Client{Timeout: c.timeout}
	} else {
		owned := *c.http
		c.http = &owned
	}
	// Redirects could replay evidence or credentials to another endpoint.
	c.http.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	if c.maxBytes <= 0 {
		c.maxBytes = JEVDefaultMaxResponseBytes
	}
	// Reserve one extra byte for detecting oversized responses without overflow.
	if c.maxBytes == 1<<63-1 {
		return nil, &JEVError{Kind: ErrJEVNotConfigured, Detail: "response size limit is too large"}
	}
	return c, nil
}

func checkJEVEndpoint(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return errors.New("endpoint is not an absolute URL")
	}
	if u.User != nil {
		return errors.New("endpoint must not contain credentials")
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		host := u.Hostname()
		if host == "localhost" {
			return nil
		}
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return nil
		}
		return errors.New("plain http endpoint is only allowed for loopback hosts")
	default:
		return errors.New("endpoint scheme must be https")
	}
}

// Evaluate sends one evaluation request and returns a schema-validated
// response. Validation covers shape and value ranges only; callers must still
// check evidence/scope/as-of before showing any result as an assessment.
func (c *JEVClient) Evaluate(ctx context.Context, req models.JEVRequest) (models.JEVResponse, error) {
	if c == nil {
		return models.JEVResponse{}, &JEVError{Kind: ErrJEVNotConfigured, Detail: "nil client"}
	}
	if req.Model == "" {
		req.Model = c.model
	}
	if err := ValidateJEVRequest(req); err != nil {
		return models.JEVResponse{}, &JEVError{Kind: ErrJEVInvalidRequest, Detail: "request does not match provider schema"}
	}
	body, err := json.Marshal(req)
	if err != nil {
		return models.JEVResponse{}, &JEVError{Kind: ErrJEVInvalidRequest, Detail: "request is not JSON-encodable"}
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return models.JEVResponse{}, &JEVError{Kind: ErrJEVNotConfigured, Detail: "cannot build request"}
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return models.JEVResponse{}, &JEVError{Kind: ErrJEVTransport, Err: c.sanitizeTransportErr(ctx, err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, jevMaxErrorDrainBytes))
		return models.JEVResponse{}, classifyJEVStatus(resp)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBytes+1))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return models.JEVResponse{}, &JEVError{Kind: ErrJEVTransport, StatusCode: resp.StatusCode, Detail: "reading response", Err: ctxErr}
		}
		return models.JEVResponse{}, &JEVError{Kind: ErrJEVTransport, StatusCode: resp.StatusCode, Detail: "reading response failed"}
	}
	if int64(len(raw)) > c.maxBytes {
		return models.JEVResponse{}, &JEVError{Kind: ErrJEVInvalidResponse, StatusCode: resp.StatusCode, Detail: fmt.Sprintf("response exceeds %d bytes", c.maxBytes)}
	}

	var out models.JEVResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return models.JEVResponse{}, &JEVError{Kind: ErrJEVInvalidResponse, StatusCode: resp.StatusCode, Detail: "malformed JSON or invalid field type"}
	}
	if err := ValidateJEVResponse(req.Questions, out); err != nil {
		return models.JEVResponse{}, &JEVError{Kind: ErrJEVInvalidResponse, StatusCode: resp.StatusCode, Detail: "response does not match provider schema"}
	}
	return out, nil
}

// sanitizeTransportErr keeps context errors matchable and strips everything
// else down to the error type, so URLs with query strings or proxy details
// never leak through logs. The key is never part of these errors.
func (c *JEVClient) sanitizeTransportErr(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return errors.New("network timeout")
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return fmt.Errorf("%T", urlErr.Err)
	}
	return fmt.Errorf("%T", err)
}

func classifyJEVStatus(resp *http.Response) error {
	e := &JEVError{StatusCode: resp.StatusCode}
	switch code := resp.StatusCode; {
	case code == http.StatusBadRequest || code == http.StatusUnprocessableEntity:
		e.Kind = ErrJEVInvalidRequest
		e.Detail = "provider rejected the request body"
	case code == http.StatusUnauthorized || code == http.StatusForbidden:
		e.Kind = ErrJEVAuthentication
		e.Detail = "check TYPESAFE_API_KEY"
	case code == http.StatusTooManyRequests:
		e.Kind = ErrJEVRateLimited
		e.RetryAfter = resp.Header.Get("Retry-After")
	case code == 529 || code >= 500:
		e.Kind = ErrJEVProviderFailure
		e.RetryAfter = resp.Header.Get("Retry-After")
	default:
		e.Kind = ErrJEVProviderFailure
		e.Detail = "unexpected status"
	}
	return e
}
