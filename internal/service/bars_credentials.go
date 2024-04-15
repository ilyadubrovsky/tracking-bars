package service

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type BarsCredentials interface {
	Authorization(ctx context.Context, credentials *domain.BarsCredentials) error
	Logout(ctx context.Context, userID int64) error
}
