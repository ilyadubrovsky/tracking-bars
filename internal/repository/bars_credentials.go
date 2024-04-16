package repository

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type BarsCredentials interface {
	Save(ctx context.Context, barsCredentials *domain.BarsCredentials) error
	GetByUserID(ctx context.Context, userID int64) (*domain.BarsCredentials, error)
	Delete(ctx context.Context, userID int64) error
	GetAll(ctx context.Context) ([]*domain.BarsCredentials, error)
	Count(ctx context.Context) (int, error)
}
