package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilyadubrovsky/tracking-bars/internal/database"
	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	"github.com/jackc/pgx/v4"
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
	query := `
		INSERT INTO users
		VALUES ($1)
		ON CONFLICT (id) DO NOTHING 
	`

	_, err := r.db.Exec(ctx, query, user.ID)
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}

func (r *repo) Get(ctx context.Context, userID int64) (*domain.User, error) {
	query := `
	SELECT 
	    id
	FROM 
	    users
	WHERE id = $1
	`

	user := new(domain.User)
	err := r.db.QueryRow(ctx, query, userID).Scan(&user.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db.QueryRow: %w", err)
	}

	return user, nil
}
