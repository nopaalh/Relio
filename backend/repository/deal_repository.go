package repository

import (
	"context"
	"time"

	"github.com/nopaalh/Relio/backend/models"
)

type DealRepository interface {
	List(ctx context.Context, asOf time.Time) ([]models.Deal, error)
	FindByID(ctx context.Context, dealID string, asOf time.Time) (models.Deal, error)
}
