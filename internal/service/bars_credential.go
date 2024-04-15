package service

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type BarsCredential interface {
	Authorization(ctx context.Context, credentials *domain.BarsCredentials) error
	Logout(ctx context.Context, userID int64) error
	GetAllAuthorized(ctx context.Context) ([]*domain.BarsCredentials, error)
}
