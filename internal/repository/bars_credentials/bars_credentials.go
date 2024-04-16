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

func (r *repo) GetByUserID(ctx context.Context, userID int64) (*domain.BarsCredentials, error) {
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
		AND deleted_at IS NULL
	`

	dboCredential := new(dbo.BarsCredentials)
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&dboCredential.UserID,
		&dboCredential.Username,
		&dboCredential.Password,
		&dboCredential.CreatedAt,
		&dboCredential.UpdatedAt,
		&dboCredential.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) || dboCredential.DeletedAt != nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db.QueryRow.Scan: %w", err)
	}

	return dbo.ToDomain(dboCredential), nil
}

func (r *repo) Delete(ctx context.Context, userID int64) error {
	query := `
		UPDATE bars_credentials
		SET 
			deleted_at = $2
		WHERE user_id = $1
	`

	_, err := r.db.Exec(ctx, query, userID, time.Now())
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}

func (r *repo) GetAll(ctx context.Context) ([]*domain.BarsCredentials, error) {
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
		WHERE deleted_at IS NULL
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("db.Query: %w", err)
	}
	defer func() {
		rows.Close()
	}()

	dboCredentials := make([]*dbo.BarsCredentials, 0)
	for rows.Next() {
		dboCredential := new(dbo.BarsCredentials)
		err = rows.Scan(
			&dboCredential.UserID,
			&dboCredential.Username,
			&dboCredential.Password,
			&dboCredential.CreatedAt,
			&dboCredential.UpdatedAt,
			&dboCredential.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		dboCredentials = append(dboCredentials, dboCredential)
	}

	return dbo.ManyToDomain(dboCredentials), nil
}
