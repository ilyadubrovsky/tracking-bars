package progress_tables

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ilyadubrovsky/tracking-bars/internal/database"
	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository/progress_tables/dbo"
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

func (r *repo) Save(ctx context.Context, progressTable *domain.ProgressTable) error {
	dboProgressTable, err := dbo.FromDomain(progressTable)
	if err != nil {
		return fmt.Errorf("dbo.FromDomain: %w", err)
	}

	query := `
		INSERT INTO progress_tables (
		    user_id, 
		    progress_table,
		    created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		SET
		    progress_table = $2,
		    updated_at = $4
	`

	now := time.Now()
	_, err = r.db.Exec(ctx, query,
		dboProgressTable.UserID,        // $1
		dboProgressTable.ProgressTable, // $2
		now,                            // $3
		now,                            // $4
	)
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}

func (r *repo) GetProgressTable(ctx context.Context, userID int64) (*domain.ProgressTable, error) {
	query := `
		SELECT 
		    user_id, 
		    progress_table, 
		    created_at, 
		    updated_at
		FROM
		    progress_tables
		WHERE user_id = $1
	`

	dboProgressTable := new(dbo.ProgressTable)
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&dboProgressTable.UserID,
		&dboProgressTable.ProgressTable,
		&dboProgressTable.CreatedAt,
		&dboProgressTable.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db.QueryRow.Scan: %w", err)
	}

	progressTable, err := dbo.ToDomain(dboProgressTable)
	if err != nil {
		return nil, fmt.Errorf("dbo.ToDomain: %w", err)
	}

	return progressTable, nil
}

func (r *repo) Delete(ctx context.Context, userID int64) error {
	query := `
		DELETE FROM progress_tables 
		WHERE user_id = $1
	`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}
