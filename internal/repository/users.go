package repository

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type Users interface {
	Save(ctx context.Context, user *domain.User) error
	GetByUserID(ctx context.Context, userID int64) (*domain.User, error)
	GetAll(ctx context.Context) ([]*domain.User, error)
	Delete(ctx context.Context, userID int64) error
}
