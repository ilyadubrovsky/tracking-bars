package grades_changes

import (
	"github.com/ilyadubrovsky/tracking-bars/internal/config"
	"github.com/ilyadubrovsky/tracking-bars/internal/service"
)

type svc struct {
	progressTableSvc service.ProgressTable
	cfg              config.Bars
}

func NewService(
	progressTableSvc service.ProgressTable,
	cfg config.Bars,
) *svc {
	return &svc{
		progressTableSvc: progressTableSvc,
		cfg:              cfg,
	}
}

func (s *svc) Start() (func(), error) {
	return nil, nil
}

func (s *svc) Stop() error {
	return nil
}
