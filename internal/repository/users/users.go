package repository

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type repo struct {
}

func NewUsers() *repo {
	return &repo{}
}

func (r *repo) Save(ctx context.Context, user *domain.User) error {
	return nil
}
