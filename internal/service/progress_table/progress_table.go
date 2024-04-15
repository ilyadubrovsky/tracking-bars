package progress_table

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository"
)

type svc struct {
	progressTablesRepo repository.ProgressTables
}

func NewService(
	progressTablesRepo repository.ProgressTables,
) *svc {
	return &svc{
		progressTablesRepo: progressTablesRepo,
	}
}

func (s *svc) Save(ctx context.Context, progressTable *domain.ProgressTable) error {
	return nil
}
