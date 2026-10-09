package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nopaalh/Relio/backend/models"
)

type stubDealRepository struct {
	deals     []models.Deal
	deal      models.Deal
	err       error
	lastID    string
	lastAsOf  time.Time
	findCalls int
}

func (r *stubDealRepository) List(ctx context.Context, asOf time.Time) ([]models.Deal, error) {
	r.lastAsOf = asOf
	return r.deals, r.err
}

func (r *stubDealRepository) FindByID(ctx context.Context, dealID string, asOf time.Time) (models.Deal, error) {
	r.lastID = dealID
	r.lastAsOf = asOf
	r.findCalls++
	return r.deal, r.err
}

func TestDealServiceList(t *testing.T) {
	asOf := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	repo := &stubDealRepository{deals: []models.Deal{{ID: "DL-001"}}}

	deals, err := NewDealService(repo).List(context.Background(), asOf)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(deals) != 1 || deals[0].ID != "DL-001" {
		t.Fatalf("unexpected deals: %+v", deals)
	}
	if !repo.lastAsOf.Equal(asOf) {
		t.Fatalf("asOf = %v, want %v", repo.lastAsOf, asOf)
	}
}

func TestDealServiceFindByID(t *testing.T) {
	asOf := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	repo := &stubDealRepository{deal: models.Deal{ID: "DL-001"}}

	deal, err := NewDealService(repo).FindByID(context.Background(), " DL-001 ", asOf)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if deal.ID != "DL-001" || repo.lastID != "DL-001" || !repo.lastAsOf.Equal(asOf) {
		t.Fatalf("unexpected repository call: deal=%+v, id=%q, asOf=%v", deal, repo.lastID, repo.lastAsOf)
	}
}

func TestDealServiceRejectsEmptyID(t *testing.T) {
	repo := &stubDealRepository{}

	_, err := NewDealService(repo).FindByID(context.Background(), " ", time.Time{})
	if !errors.Is(err, ErrInvalidDealID) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidDealID)
	}
	if repo.findCalls != 0 {
		t.Fatal("repository must not be called for an empty deal ID")
	}
}

func TestDealServicePreservesRepositoryError(t *testing.T) {
	want := errors.New("repository unavailable")
	repo := &stubDealRepository{err: want}

	_, err := NewDealService(repo).List(context.Background(), time.Time{})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}
