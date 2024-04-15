package service

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type Grade interface {
	Save(ctx context.Context, progressTable *domain.ProgressTable) error
}
