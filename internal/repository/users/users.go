package users

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/database"
	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type repo struct {
	db database.PG
}

func NewRepository(db database.PG) *repo {
	return &repo{
		db: db,
	}
}

func (r *repo) Save(ctx context.Context, user *domain.User) error {
	return nil
}
