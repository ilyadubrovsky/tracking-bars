package service

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type User interface {
	Save(ctx context.Context, user *domain.User) error
}
