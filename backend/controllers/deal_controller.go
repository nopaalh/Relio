package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
	"github.com/nopaalh/Relio/backend/services"
)

const snapshotDate = "2026-10-01"

var errUnsupportedSnapshot = errors.New("as_of must be 2026-10-01; historical queries are not configured yet")

type DealReader interface {
	List(ctx context.Context, asOf time.Time) ([]models.Deal, error)
	FindByID(ctx context.Context, dealID string, asOf time.Time) (models.Deal, error)
}

type DealController struct {
	deals DealReader
}

func NewDealController(deals DealReader) *DealController {
	return &DealController{deals: deals}
}

func (c *DealController) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{Code: "method_not_allowed", Message: "only GET is supported"})
		return
	}

	asOf, err := requestSnapshot(r)
	if err != nil {
		writeDealError(w, err)
		return
	}

	deals, err := c.deals.List(r.Context(), asOf)
	if err != nil {
		writeDealError(w, err)
		return
	}
	if deals == nil {
		deals = []models.Deal{}
	}
	writeJSON(w, http.StatusOK, deals)
}

func (c *DealController) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, models.ErrorResponse{Code: "method_not_allowed", Message: "only GET is supported"})
		return
	}

	asOf, err := requestSnapshot(r)
	if err != nil {
		writeDealError(w, err)
		return
	}
	dealID := strings.TrimSpace(r.PathValue("deal_id"))
	if dealID == "" {
		writeDealError(w, services.ErrInvalidDealID)
		return
	}

	deal, err := c.deals.FindByID(r.Context(), dealID, asOf)
	if err != nil {
		writeDealError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, deal)
}

func requestSnapshot(r *http.Request) (time.Time, error) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return time.Time{}, errUnsupportedSnapshot
	}
	if values, exists := query["as_of"]; exists && (len(values) != 1 || values[0] != snapshotDate) {
		return time.Time{}, errUnsupportedSnapshot
	}

	// This is a snapshot date token, not an agreed historical end-of-day cutoff.
	return time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC), nil
}

func writeDealError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	body := models.ErrorResponse{Code: "internal_error", Message: "could not retrieve deal data"}
	switch {
	case errors.Is(err, services.ErrInvalidDealID):
		status = http.StatusBadRequest
		body = models.ErrorResponse{Code: "invalid_deal_id", Message: "deal ID is required"}
	case errors.Is(err, errUnsupportedSnapshot):
		status = http.StatusBadRequest
		body = models.ErrorResponse{Code: "invalid_as_of", Message: errUnsupportedSnapshot.Error()}
	case errors.Is(err, repository.ErrDealNotFound):
		status = http.StatusNotFound
		body = models.ErrorResponse{Code: "deal_not_found", Message: "deal not found"}
	case errors.Is(err, repository.ErrDealDataUnavailable):
		status = http.StatusServiceUnavailable
		body = models.ErrorResponse{Code: "deal_data_unavailable", Message: "deal database adapter is not configured"}
	}
	writeJSON(w, status, body)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	payload, err := json.Marshal(body)
	if err != nil {
		status = http.StatusInternalServerError
		payload = []byte(`{"code":"internal_error","message":"could not encode response"}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(payload, '\n'))
}
