package repository

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type GradesChangesOutbox interface {
	GradesChanges(ctx context.Context, limit int64) ([]*domain.GradeChange, error)
	Delete(ctx context.Context, ids []int64) error
}
