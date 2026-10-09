package repository

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

type contextCursor struct {
	Context string `json:"context"`
	Filter  string `json:"filter"`
	After   string `json:"after"`
}

func filterKey(parts ...any) string {
	b, _ := json.Marshal(parts)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func cursorAfter(token *string, contextID, filter string) (string, error) {
	if token == nil {
		return "", nil
	}
	if len(*token) > 4096 {
		return "", &RepositoryError{Code: InvalidQuery}
	}
	b, err := base64.RawURLEncoding.DecodeString(*token)
	if err != nil {
		return "", &RepositoryError{Code: InvalidQuery}
	}
	var c contextCursor
	if err = json.Unmarshal(b, &c); err != nil || c.After == "" {
		return "", &RepositoryError{Code: InvalidQuery}
	}
	if c.Context != contextID {
		return "", &RepositoryError{Code: ContextMismatch}
	}
	if c.Filter != filter {
		return "", &RepositoryError{Code: InvalidQuery}
	}
	return c.After, nil
}
func nextCursor(contextID, filter, after string) *string {
	b, _ := json.Marshal(contextCursor{contextID, filter, after})
	s := base64.RawURLEncoding.EncodeToString(b)
	return &s
}
func sortedFilter(ids []string) ([]string, error) {
	out := append([]string{}, ids...)
	for _, id := range out {
		if id == "" || strings.TrimSpace(id) != id {
			return nil, &RepositoryError{Code: InvalidQuery}
		}
	}
	sort.Strings(out)
	for i := 1; i < len(out); i++ {
		if out[i] == out[i-1] {
			return nil, &RepositoryError{Code: InvalidQuery}
		}
	}
	return out, nil
}
func matches(values []string, value string) bool {
	if len(values) == 0 {
		return true
	}
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
