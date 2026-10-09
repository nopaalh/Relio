package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
)

var ErrInvalidDealID = errors.New("deal ID is required")

type DealService struct {
	deals repository.DealRepository
}

func NewDealService(deals repository.DealRepository) *DealService {
	return &DealService{deals: deals}
}

func (s *DealService) List(ctx context.Context, asOf time.Time) ([]models.Deal, error) {
	return s.deals.List(ctx, asOf)
}

func (s *DealService) FindByID(ctx context.Context, dealID string, asOf time.Time) (models.Deal, error) {
	dealID = strings.TrimSpace(dealID)
	if dealID == "" {
		return models.Deal{}, ErrInvalidDealID
	}

	return s.deals.FindByID(ctx, dealID, asOf)
}
