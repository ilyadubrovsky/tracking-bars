package repository

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type ProgressTables interface {
	Save(ctx context.Context, progressTable *domain.ProgressTable) error
	GetByUserID(ctx context.Context, userID int64) (*domain.ProgressTable, error)
	Delete(ctx context.Context, userID int64) error
}
