package bars_credentials

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ilyadubrovsky/tracking-bars/internal/database"
	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository/bars_credentials/dbo"
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

func (r *repo) Save(ctx context.Context, barsCredentials *domain.BarsCredentials) error {
	query := `
		INSERT INTO bars_credentials (
		    user_id,
		    username,
		    password,
		    created_at,
		    updated_at
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE
		SET
		    username = $2,
		    password = $3,
		    updated_at = $5,
			deleted_at = $6
	`

	now := time.Now()
	_, err := r.db.Exec(ctx, query,
		barsCredentials.UserID,   // $1
		barsCredentials.Username, // $2
		barsCredentials.Password, // $3
		now,                      // $4
		now,                      // $5
		nil,                      // $6
	)
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}

func (r *repo) Get(ctx context.Context, userID int64) (*domain.BarsCredentials, error) {
	query := `
		SELECT 
			user_id, 
			username, 
			password, 
			created_at,
			updated_at,
			deleted_at
		FROM
		    bars_credentials
		WHERE user_id = $1
	`

	dboCredentials := new(dbo.BarsCredentials)
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&dboCredentials.UserID,
		&dboCredentials.Username,
		&dboCredentials.Password,
		&dboCredentials.CreatedAt,
		&dboCredentials.UpdatedAt,
		&dboCredentials.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) || dboCredentials.DeletedAt != nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db.QueryRow.Scan: %w", err)
	}

	return dboCredentials.ToDomain(), nil
}

func (r *repo) Delete(ctx context.Context, userID int64) error {
	query := `
		UPDATE bars_credentials
		SET 
		    password = $2,
			deleted_at = $3
		WHERE user_id = $1
	`

	_, err := r.db.Exec(ctx, query, userID, []byte{}, time.Now())
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}
