package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"github.com/nopaalh/Relio/backend/models"
)

const (
	ContractVersion             = "context-contract-v0.1-candidate"
	CalendarZone                = "Asia/Jakarta"
	MaxAsOf         models.Date = "2026-10-01"
)

// NewSnapshotContext is for trusted server code. It does not authorize a caller
// or verify that the supplied dataset version is the active published dataset.
// HTTP defaults and access construction belong to O2; readiness belongs to the adapter.
func NewSnapshotContext(asOf models.Date, datasetVersion string, access models.AccessScope) (models.SnapshotContext, error) {
	if _, err := models.ParseDate(string(asOf)); err != nil {
		return models.SnapshotContext{}, &RepositoryError{Code: InvalidSnapshot, Cause: err}
	}
	if asOf > MaxAsOf {
		return models.SnapshotContext{}, &RepositoryError{Code: InvalidSnapshot}
	}
	if strings.TrimSpace(datasetVersion) == "" {
		return models.SnapshotContext{}, &RepositoryError{Code: DataNotReady}
	}
	if strings.TrimSpace(datasetVersion) != datasetVersion {
		return models.SnapshotContext{}, &RepositoryError{Code: InvalidSnapshot}
	}
	accounts, err := canonicalIDs(access.AllowedAccountIDs)
	if err != nil {
		return models.SnapshotContext{}, err
	}
	deals, err := canonicalIDs(access.AllowedDealIDs)
	if err != nil {
		return models.SnapshotContext{}, err
	}
	analogs, err := canonicalIDs(access.AllowedAnalogAccountIDs)
	if err != nil {
		return models.SnapshotContext{}, err
	}
	snapshot := models.SnapshotContext{
		AsOf: asOf, DatasetVersion: datasetVersion, ContractVersion: ContractVersion, CalendarZone: CalendarZone,
		Access: models.AccessScope{AllowedAccountIDs: accounts, AllowedDealIDs: deals, AllowedAnalogAccountIDs: analogs, AllowCompanyEvidence: access.AllowCompanyEvidence},
	}
	// Fixed-order JSON, canonical sets, and explicit booleans avoid delimiter and
	// nil-vs-empty/order collisions. This identifier is not an access token.
	key := struct {
		AsOf            models.Date
		DatasetVersion  string
		ContractVersion string
		CalendarZone    string
		Access          models.AccessScope
	}{snapshot.AsOf, snapshot.DatasetVersion, snapshot.ContractVersion, snapshot.CalendarZone, snapshot.Access}
	encoded, _ := json.Marshal(key) // This fixed struct contains only strings, lists, and bool.
	digest := sha256.Sum256(encoded)
	snapshot.ContextID = "ctx:" + hex.EncodeToString(digest[:])
	return snapshot, nil
}

// ValidateSnapshot checks context self-consistency. The database adapter must
// additionally enforce active dataset readiness/version and all result/ref scope.
func ValidateSnapshot(snapshot models.SnapshotContext) error {
	if snapshot.ContractVersion != ContractVersion || snapshot.CalendarZone != CalendarZone {
		return &RepositoryError{Code: ContextMismatch}
	}
	expected, err := NewSnapshotContext(snapshot.AsOf, snapshot.DatasetVersion, snapshot.Access)
	if err != nil {
		return err
	}
	if snapshot.ContextID != expected.ContextID {
		return &RepositoryError{Code: ContextMismatch}
	}
	return nil
}

func canonicalIDs(ids []string) ([]string, error) {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" || strings.TrimSpace(id) != id {
			return nil, &RepositoryError{Code: InvalidQuery}
		}
		result = append(result, id)
	}
	sort.Strings(result)
	unique := result[:0]
	for _, id := range result {
		if len(unique) == 0 || unique[len(unique)-1] != id {
			unique = append(unique, id)
		}
	}
	return unique, nil
}
