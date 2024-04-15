package repository

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type BarsCredentials interface {
	Save(ctx context.Context, barsCredentials *domain.BarsCredentials) error
	Get(ctx context.Context, userID int64) (*domain.BarsCredentials, error)
	Delete(ctx context.Context, userID int64) error
	GetAllAuthorized(ctx context.Context) ([]*domain.BarsCredentials, error)
}
