package service

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	"github.com/ilyadubrovsky/tracking-bars/pkg/bars"
)

type Bars interface {
	Authorization(ctx context.Context, credentials *domain.BarsCredentials) error
	Logout(ctx context.Context, userID int64) error
	GetProgressTable(
		ctx context.Context,
		credentials *domain.BarsCredentials,
		barsClient bars.Client,
	) (*domain.ProgressTable, error)
}
