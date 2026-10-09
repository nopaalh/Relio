package repository

import (
	"context"
	"errors"
	"time"

	"github.com/nopaalh/Relio/backend/models"
)

var (
	ErrDealNotFound        = errors.New("deal not found")
	ErrDealDataUnavailable = errors.New("deal database adapter is not configured")
)

type DealRepository interface {
	List(ctx context.Context, asOf time.Time) ([]models.Deal, error)
	FindByID(ctx context.Context, dealID string, asOf time.Time) (models.Deal, error)
}
