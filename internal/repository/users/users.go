package users

import (
	"context"
	"errors"
	"fmt"
	"time"

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
		INSERT INTO users (
			id,
		    created_at
		)
		VALUES ($1, $2)
		ON CONFLICT (id)
		DO UPDATE
		SET deleted_at = NULL;
	`

	_, err := r.db.Exec(ctx, query, user.ID, time.Now())
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}

func (r *repo) GetByUserID(ctx context.Context, userID int64) (*domain.User, error) {
	query := `
	SELECT 
	    id
	FROM 
	    users
	WHERE id = $1
	AND deleted_at IS NULL
	`

	user := new(domain.User)
	err := r.db.QueryRow(ctx, query, userID).Scan(&user.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db.QueryRow.Scan: %w", err)
	}

	return user, nil
}

func (r *repo) GetAll(ctx context.Context) ([]*domain.User, error) {
	query := `
	SELECT 
	    id
	FROM 
	    users
	WHERE deleted_at IS NULL
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("db.Query: %w", err)
	}
	defer func() {
		rows.Close()
	}()

	users := make([]*domain.User, 0)
	for rows.Next() {
		user := new(domain.User)
		err = rows.Scan(
			&user.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *repo) Delete(ctx context.Context, userID int64) error {
	query := `
		UPDATE users
		SET deleted_at = $2
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, userID, time.Now())
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}
