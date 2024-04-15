package service

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type User interface {
	Save(ctx context.Context, user *domain.User) error
	Get(ctx context.Context, userID int64) (*domain.User, error)
}
