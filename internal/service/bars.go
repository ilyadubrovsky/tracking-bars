package service

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	"github.com/ilyadubrovsky/tracking-bars/pkg/bars"
)

type Bars interface {
	Authorization(
		ctx context.Context,
		userID int64,
		username string,
		password []byte,
	) error
	Logout(ctx context.Context, userID int64) error
	GetProgressTable(
		ctx context.Context,
		username string,
		password []byte,
		barsClient bars.Client,
	) (*domain.ProgressTable, error)
}
