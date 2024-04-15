package bars_credentials

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

func (r *repo) Save(ctx context.Context, barsCredentials *domain.BarsCredentials) error {
	return nil
}

func (r *repo) Get(ctx context.Context, userID int64) (*domain.BarsCredentials, error) {
	return nil, nil
}

func (r *repo) Delete(ctx context.Context, userID int64) error {
	return nil
}
