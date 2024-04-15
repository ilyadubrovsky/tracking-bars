package users

import (
	"context"
	"fmt"

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
	query, vals, err := buildSaveOneQuery(user)
	if err != nil {
		return fmt.Errorf("buildSaveOneQuery: %w", err)
	}

	_, err = r.db.Exec(ctx, query, vals...)
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}

func (r *repo) Get(ctx context.Context, userID int64) (*domain.User, error) {
	return nil, nil
}

func buildSaveOneQuery(user *domain.User) (string, []interface{}, error) {
	query := `
		INSERT INTO users
		VALUES ($1)
		ON CONFLICT (id) DO NOTHING 
	`

	//row, err := dbo.FromDomainToDBO(user)
	//if err != nil {
	//	return "", nil, fmt.Errorf("dbo.FromDomainToDBO (user): %w", err)
	//}

	return query, []interface{}{user.ID}, nil
}
