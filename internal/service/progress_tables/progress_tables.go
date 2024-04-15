package progress_tables

import "github.com/ilyadubrovsky/tracking-bars/internal/repository"

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
