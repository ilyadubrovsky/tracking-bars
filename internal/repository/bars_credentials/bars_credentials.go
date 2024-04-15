package repository

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type repo struct {
}

func NewBarsCredentials() *repo {
	return &repo{}
}

func (r *repo) Save(ctx context.Context, barsCredentials *domain.BarsCredentials) error {
	return nil
}

func (r *repo) Get(ctx context.Context, userID int64) (*domain.BarsCredentials, error) {
	return nil, nil
}

func (r *repo) Delete(ctx context.Context, userID int64) error {
	return nil
}
