package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
)

const (
	contextQueryTimeout = 120 * time.Second
	contextCursorLimit  = 4096
	contextIDLimit      = 256
	contextBodyLimit    = 16 << 10
)

// ContextReader supplies server-authorized snapshots and canonical wire envelopes.
type ContextReader interface {
	Snapshot(asOf models.Date) (models.SnapshotContext, error)
	List(context.Context, models.SnapshotContext, models.DealListOptions) (models.DealFactsPage, error)
	Deal(context.Context, string, models.SnapshotContext) (models.DealFactsResult, error)
	Graph(context.Context, string, models.SnapshotContext, models.GraphOptions) (models.GraphResult, error)
	Timeline(context.Context, string, models.SnapshotContext, models.TimelineOptions) (models.TimelineResult, error)
	Evidence(context.Context, string, models.SnapshotContext) (models.EvidenceResult, error)
}

type ContextController struct {
	reader ContextReader
}

func NewContextController(reader ContextReader) *ContextController {
	return &ContextController{reader: reader}
}

func (c *ContextController) List(w http.ResponseWriter, r *http.Request) {
	query, asOf, ok := contextRequest(w, r, http.MethodGet, "limit", "cursor", "account_ids", "deal_types")
	if !ok {
		return
	}
	limit, cursor, err := contextPage(query)
	options := models.DealListOptions{Limit: limit, Cursor: cursor}
	if err == nil {
		err = contextFilters(query, map[string]*[]string{"account_ids": &options.AccountIDs, "deal_types": &options.DealTypes})
	}
	if err != nil {
		WriteContextError(w, err)
		return
	}
	c.respond(w, r, asOf, func(ctx context.Context, snapshot models.SnapshotContext) (any, error) {
		return c.reader.List(ctx, snapshot, options)
	})
}

func (c *ContextController) Get(w http.ResponseWriter, r *http.Request) {
	_, asOf, ok := contextRequest(w, r, http.MethodGet)
	if !ok || !contextPathID(w, r, "deal_id") {
		return
	}
	c.respond(w, r, asOf, func(ctx context.Context, snapshot models.SnapshotContext) (any, error) {
		return c.reader.Deal(ctx, r.PathValue("deal_id"), snapshot)
	})
}

func (c *ContextController) Graph(w http.ResponseWriter, r *http.Request) {
	query, asOf, ok := contextRequest(w, r, http.MethodGet, "depth", "max_nodes", "max_edges", "focus_event_id")
	if !ok || !contextPathID(w, r, "deal_id") {
		return
	}
	options := models.GraphOptions{}
	var err error
	options.Depth, err = contextInteger(query, "depth", 1, 2)
	if err == nil {
		options.MaxNodes, err = contextInteger(query, "max_nodes", 150, 150)
	}
	if err == nil {
		options.MaxEdges, err = contextInteger(query, "max_edges", 300, 300)
	}
	if values, present := query["focus_event_id"]; present && err == nil {
		if !validContextID(values[0]) {
			err = invalidContextQuery()
		} else {
			options.FocusEventID = &values[0]
		}
	}
	if err != nil {
		WriteContextError(w, err)
		return
	}
	c.respond(w, r, asOf, func(ctx context.Context, snapshot models.SnapshotContext) (any, error) {
		return c.reader.Graph(ctx, r.PathValue("deal_id"), snapshot, options)
	})
}

func (c *ContextController) Timeline(w http.ResponseWriter, r *http.Request) {
	query, asOf, ok := contextRequest(w, r, http.MethodGet, "limit", "cursor", "event_ids", "event_types", "actor_node_ids", "statuses")
	if !ok || !contextPathID(w, r, "deal_id") {
		return
	}
	limit, cursor, err := contextPage(query)
	options := models.TimelineOptions{Limit: limit, Cursor: cursor}
	if err == nil {
		err = contextFilters(query, map[string]*[]string{
			"event_ids": &options.EventIDs, "event_types": &options.EventTypes,
			"actor_node_ids": &options.ActorNodeIDs, "statuses": &options.Statuses,
		})
	}
	if err != nil {
		WriteContextError(w, err)
		return
	}
	c.respond(w, r, asOf, func(ctx context.Context, snapshot models.SnapshotContext) (any, error) {
		return c.reader.Timeline(ctx, r.PathValue("deal_id"), snapshot, options)
	})
}

func (c *ContextController) Evidence(w http.ResponseWriter, r *http.Request) {
	_, asOf, ok := contextRequest(w, r, http.MethodGet)
	if !ok || !contextPathID(w, r, "evidence_id") {
		return
	}
	c.respond(w, r, asOf, func(ctx context.Context, snapshot models.SnapshotContext) (any, error) {
		return c.reader.Evidence(ctx, r.PathValue("evidence_id"), snapshot)
	})
}

func (c *ContextController) Assessment(w http.ResponseWriter, r *http.Request) {
	_, asOf, ok := contextRequest(w, r, http.MethodGet)
	if !ok || !contextPathID(w, r, "deal_id") {
		return
	}
	assessor, ok := c.reader.(interface {
		Assessment(context.Context, string, models.SnapshotContext) (models.DealAssessment, error)
	})
	if !ok {
		c.unavailable(w, r, asOf, r.PathValue("deal_id"), "assessment_unavailable", "deal assessment is not available")
		return
	}
	c.respond(w, r, asOf, func(ctx context.Context, snapshot models.SnapshotContext) (any, error) {
		return assessor.Assessment(ctx, r.PathValue("deal_id"), snapshot)
	})
}

func (c *ContextController) ActionCandidates(w http.ResponseWriter, r *http.Request) {
	query, asOf, ok := contextRequest(w, r, http.MethodGet, "limit", "cursor")
	if !ok || !contextPathID(w, r, "deal_id") {
		return
	}
	limit, cursor, err := contextPage(query)
	if err != nil {
		WriteContextError(w, err)
		return
	}
	reader, ok := c.reader.(interface {
		ActionCandidates(context.Context, string, models.SnapshotContext, models.CandidateOptions) (models.ActionCandidatesResult, error)
	})
	if !ok {
		c.unavailable(w, r, asOf, r.PathValue("deal_id"), "action_candidates_unavailable", "action candidates are not available")
		return
	}
	c.respond(w, r, asOf, func(ctx context.Context, snapshot models.SnapshotContext) (any, error) {
		return reader.ActionCandidates(ctx, r.PathValue("deal_id"), snapshot, models.CandidateOptions{RelevanceRuleVersion: repository.FullRelevanceVersion, Limit: limit, Cursor: cursor})
	})
}

func (c *ContextController) Compare(w http.ResponseWriter, r *http.Request) {
	query, asOf, ok := contextRequest(w, r, http.MethodPost)
	if !ok || !contextPathID(w, r, "deal_id") {
		return
	}
	body, ok := contextBody(w, r, "action_ids")
	if !ok {
		return
	}
	asOf, err := contextBodyDate(query, body, asOf)
	if err != nil {
		WriteContextError(w, err)
		return
	}
	var ids []string
	if json.Unmarshal(body["action_ids"], &ids) != nil || len(ids) < 2 || len(ids) > 4 {
		WriteContextError(w, invalidContextQuery())
		return
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !validContextID(id) || seen[id] {
			WriteContextError(w, invalidContextQuery())
			return
		}
		seen[id] = true
	}
	reader, ok := c.reader.(interface {
		CompareActions(context.Context, string, models.SnapshotContext, []string, string) (models.ActionComparisonResult, error)
	})
	if !ok {
		c.unavailable(w, r, asOf, r.PathValue("deal_id"), "comparison_unavailable", "action comparison is not available; no scores or rankings were produced")
		return
	}
	c.respond(w, r, asOf, func(ctx context.Context, snapshot models.SnapshotContext) (any, error) {
		return reader.CompareActions(ctx, r.PathValue("deal_id"), snapshot, ids, repository.FullRelevanceVersion)
	})
}

func (c *ContextController) Ask(w http.ResponseWriter, r *http.Request) {
	query, asOf, ok := contextRequest(w, r, http.MethodPost)
	if !ok {
		return
	}
	body, ok := contextBody(w, r, "deal_id", "question")
	if !ok {
		return
	}
	asOf, err := contextBodyDate(query, body, asOf)
	if err != nil {
		WriteContextError(w, err)
		return
	}
	var dealID, question string
	if json.Unmarshal(body["deal_id"], &dealID) != nil || !validContextID(dealID) ||
		json.Unmarshal(body["question"], &question) != nil || strings.TrimSpace(question) == "" || utf8.RuneCountInString(question) > 2000 {
		WriteContextError(w, invalidContextQuery())
		return
	}
	c.unavailable(w, r, asOf, dealID, "copilot_unavailable", "deal-scoped copilot retrieval and answer generation are not available")
}

func (c *ContextController) unavailableGet(w http.ResponseWriter, r *http.Request, code, message string) {
	_, asOf, ok := contextRequest(w, r, http.MethodGet)
	if !ok || !contextPathID(w, r, "deal_id") {
		return
	}
	c.unavailable(w, r, asOf, r.PathValue("deal_id"), code, message)
}

func (c *ContextController) unavailable(w http.ResponseWriter, r *http.Request, asOf models.Date, dealID, code, message string) {
	_, err := c.query(r, asOf, func(ctx context.Context, snapshot models.SnapshotContext) (any, error) {
		return c.reader.Deal(ctx, dealID, snapshot)
	})
	if err != nil {
		WriteContextError(w, err)
		return
	}
	writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Code: code, Message: message})
}

func (c *ContextController) respond(w http.ResponseWriter, r *http.Request, asOf models.Date, read func(context.Context, models.SnapshotContext) (any, error)) {
	result, err := c.query(r, asOf, read)
	if err != nil {
		WriteContextError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (c *ContextController) query(r *http.Request, asOf models.Date, read func(context.Context, models.SnapshotContext) (any, error)) (any, error) {
	ctx, cancel := context.WithTimeout(r.Context(), contextQueryTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.reader == nil {
		return nil, &repository.RepositoryError{Code: repository.DataNotReady}
	}
	snapshot, err := c.reader.Snapshot(asOf)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	result, err := read(ctx, snapshot)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return result, err
}

func contextRequest(w http.ResponseWriter, r *http.Request, method string, fields ...string) (url.Values, models.Date, bool) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != method {
		w.Header().Set("Allow", method)
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{Code: "method_not_allowed", Message: "request method is not supported"})
		return nil, "", false
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		WriteContextError(w, invalidContextQuery())
		return nil, "", false
	}
	for key, values := range query {
		if key == "as_of" {
			if len(values) != 1 {
				WriteContextError(w, &repository.RepositoryError{Code: repository.InvalidSnapshot})
				return nil, "", false
			}
		} else if !slices.Contains(fields, key) || len(values) != 1 {
			// An allowlist also prevents browser-supplied access scope or context IDs.
			WriteContextError(w, invalidContextQuery())
			return nil, "", false
		}
	}
	asOf := repository.MaxAsOf
	if values, present := query["as_of"]; present {
		asOf, err = contextDate(values[0])
		if err != nil {
			WriteContextError(w, err)
			return nil, "", false
		}
	}
	return query, asOf, true
}

func contextDate(raw string) (models.Date, error) {
	date, err := models.ParseDate(raw)
	if err != nil || date > repository.MaxAsOf {
		return "", &repository.RepositoryError{Code: repository.InvalidSnapshot}
	}
	return date, nil
}

func contextPathID(w http.ResponseWriter, r *http.Request, key string) bool {
	if !validContextID(r.PathValue(key)) {
		WriteContextError(w, invalidContextQuery())
		return false
	}
	return true
}

func validContextToken(raw string, max int) bool {
	return raw != "" && len(raw) <= max && utf8.ValidString(raw) && strings.TrimSpace(raw) == raw &&
		strings.IndexFunc(raw, unicode.IsControl) < 0
}

func validContextID(raw string) bool {
	return validContextToken(raw, contextIDLimit) && strings.IndexFunc(raw, unicode.IsSpace) < 0 && !strings.ContainsAny(raw, "/\\")
}

func invalidContextQuery() error {
	return &repository.RepositoryError{Code: repository.InvalidQuery}
}

func contextInteger(query url.Values, key string, defaultValue, maximum int) (int, error) {
	values, present := query[key]
	if !present {
		return defaultValue, nil
	}
	for _, char := range values[0] {
		if char < '0' || char > '9' {
			return 0, invalidContextQuery()
		}
	}
	value, err := strconv.Atoi(values[0])
	if err != nil || value < 1 || value > maximum {
		return 0, invalidContextQuery()
	}
	return value, nil
}

func contextPage(query url.Values) (int, *string, error) {
	limit, err := contextInteger(query, "limit", 50, 50)
	if err != nil {
		return 0, nil, err
	}
	if values, present := query["cursor"]; present {
		if !validContextToken(values[0], contextCursorLimit) {
			return 0, nil, invalidContextQuery()
		}
		return limit, &values[0], nil
	}
	return limit, nil, nil
}

func contextFilters(query url.Values, fields map[string]*[]string) error {
	for key, target := range fields {
		if values, present := query[key]; present {
			if !validContextToken(values[0], contextIDLimit) {
				return invalidContextQuery()
			}
			*target = values
		}
	}
	return nil
}

func contextBody(w http.ResponseWriter, r *http.Request, fields ...string) (map[string]json.RawMessage, bool) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeJSON(w, http.StatusUnsupportedMediaType, models.ErrorResponse{Code: "unsupported_media_type", Message: "request body must use application/json"})
		return nil, false
	}
	body, err := decodeContextBody(http.MaxBytesReader(w, r.Body, contextBodyLimit), fields)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeJSON(w, http.StatusRequestEntityTooLarge, models.ErrorResponse{Code: "request_too_large", Message: "request body exceeds the allowed size"})
		} else {
			WriteContextError(w, invalidContextQuery())
		}
		return nil, false
	}
	return body, true
}

func decodeContextBody(reader io.ReadCloser, fields []string) (map[string]json.RawMessage, error) {
	defer reader.Close()
	decoder := json.NewDecoder(reader)
	opening, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if opening != json.Delim('{') {
		return nil, invalidContextQuery()
	}
	body := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok || (key != "as_of" && !slices.Contains(fields, key)) {
			return nil, invalidContextQuery()
		}
		if _, duplicate := body[key]; duplicate {
			return nil, invalidContextQuery()
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		body[key] = value
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = invalidContextQuery()
		}
		return nil, err
	}
	return body, nil
}

func contextBodyDate(query url.Values, body map[string]json.RawMessage, defaultDate models.Date) (models.Date, error) {
	value, present := body["as_of"]
	if !present {
		return defaultDate, nil
	}
	var raw string
	if _, duplicate := query["as_of"]; duplicate || json.Unmarshal(value, &raw) != nil {
		return "", &repository.RepositoryError{Code: repository.InvalidSnapshot}
	}
	return contextDate(raw)
}
