package progress_table

import (
	"context"
	"fmt"

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
	if err := s.progressTablesRepo.Save(ctx, progressTable); err != nil {
		return fmt.Errorf("progressTablesRepo.Save: %w", err)
	}

	return nil
}

func (s *svc) GetProgressTable(ctx context.Context, userID int64) (*domain.ProgressTable, error) {
	progressTable, err := s.progressTablesRepo.GetProgressTable(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("progressTablesRepo.GetProgressTable: %w", err)
	}

	return progressTable, nil
}
