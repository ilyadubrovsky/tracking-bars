package progress_table

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository"
	"github.com/ilyadubrovsky/tracking-bars/internal/service"
)

type svc struct {
	barsCredentialSvc  service.BarsCredential
	progressTablesRepo repository.ProgressTables
}

func NewService(
	barsCredentialSvc service.BarsCredential,
	progressTablesRepo repository.ProgressTables,
) *svc {
	return &svc{
		barsCredentialSvc:  barsCredentialSvc,
		progressTablesRepo: progressTablesRepo,
	}
}

func (s *svc) Save(ctx context.Context, progressTable *domain.ProgressTable) error {
	return nil
}

func (s *svc) GetProgressTable(ctx context.Context, userID int64) (*domain.ProgressTable, error) {
	return nil, nil
}
