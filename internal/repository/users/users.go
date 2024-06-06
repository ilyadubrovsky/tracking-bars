package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ilyadubrovsky/tracking-bars/internal/database"
	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository/users/dbo"
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

// TODO тут порядок надо навести
func (r *repo) Save(ctx context.Context, user *domain.User) error {
	insertUserQuery := `
		INSERT INTO users (
			id,
		    created_at
		)
		VALUES ($1, $2)
		ON CONFLICT (id)
		DO UPDATE
		SET deleted_at = NULL;
	`

	insertBarsCredentialsQuery := `
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

	insertProgressTableQuery := `
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

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db.Begin: %w", err)
	}
	defer tx.Rollback(ctx)

	timeNow := time.Now()
	_, err = tx.Exec(
		ctx,
		insertUserQuery,
		user.ID, // $1
		timeNow, // $2
	)
	if err != nil {
		return fmt.Errorf("db.Exec insertUserQuery: %w", err)
	}

	if user.BarsCredentials != nil {
		_, err = tx.Exec(
			ctx,
			insertBarsCredentialsQuery,
			user.ID,                       // $1
			user.BarsCredentials.Username, // $2
			user.BarsCredentials.Password, // $3
			timeNow,                       // $4
			timeNow,                       // $5
			nil,                           // $6
		)
		if err != nil {
			return fmt.Errorf("tx.Exec insertBarsCredentialsQuery: %w", err)
		}
	}

	if user.ProgressTable != nil {
		progressTableDBO, err := dbo.ProgressTableFromDomain(user.ProgressTable)
		if err != nil {
			return fmt.Errorf("dbo.ProgressTableFromDomain: %w", err)
		}
		_, err = tx.Exec(
			ctx,
			insertProgressTableQuery,
			user.ID,          // $1
			progressTableDBO, // $2
			timeNow,          // $3
			timeNow,          // $4
		)
		if err != nil {
			return fmt.Errorf("tx.Exec insertProgressTableQuery: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}

	return nil
}

func (r *repo) User(ctx context.Context, userID int64) (*domain.User, error) {
	query := `
	SELECT
	  u.id,
	  bc.username,
	  bc.password,
	  pt.progress_table
	FROM users AS u
	LEFT JOIN bars_credentials AS bc
	  ON u.id = bc.user_id
	LEFT JOIN progress_tables AS pt
	  ON u.id = pt.user_id
	WHERE id = $1
	AND u.deleted_at IS NULL;
	`

	row := &dbo.UserGetRow{}
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&row.ID,
		&row.Username,
		&row.Password,
		&row.ProgressTable,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db.QueryRow.Scan: %w", err)
	}

	user, err := row.ToDomain()
	if err != nil {
		return nil, fmt.Errorf("row.ToDomain: %w", err)
	}

	return user, nil
}

// TODO add options
func (r *repo) Users(ctx context.Context) ([]*domain.User, error) {
	query := `
	SELECT
	  u.id,
	  bc.username,
	  bc.password,
	  pt.progress_table
	FROM users AS u
	LEFT JOIN bars_credentials AS bc
	  ON u.id = bc.user_id
	LEFT JOIN progress_tables AS pt
	  ON u.id = pt.user_id
	WHERE u.deleted_at IS NULL
	AND bc.deleted_at iS NULL;
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
		row := &dbo.UserGetRow{}
		err = rows.Scan(
			&row.ID,
			&row.Username,
			&row.Password,
			&row.ProgressTable,
		)
		if err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}

		user, err := row.ToDomain()
		if err != nil {
			return nil, fmt.Errorf("row.ToDomain: %w", err)
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}

	return users, nil
}

func (r *repo) Delete(ctx context.Context, userID int64) error {
	deleteProgressTableQuery := `
		DELETE FROM progress_tables 
		WHERE user_id = $1
	`

	deleteBarsCredentialsQuery := `
		UPDATE bars_credentials
		SET deleted_at = $2
		WHERE user_id = $1
	`

	deleteUserQuery := `
		UPDATE users
		SET deleted_at = $2
		WHERE id = $1
	`

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db.Begin: %w", err)
	}
	defer tx.Rollback(ctx)

	timeNow := time.Now()
	_, err = tx.Exec(
		ctx,
		deleteProgressTableQuery,
		userID, // $1
	)
	if err != nil {
		return fmt.Errorf("tx.Exec deleteProgressTableQuery: %w", err)
	}

	_, err = tx.Exec(
		ctx,
		deleteBarsCredentialsQuery,
		userID,  // $1
		timeNow, // $2
	)
	if err != nil {
		return fmt.Errorf("tx.Exec deleteBarsCredentialsQuery: %w", err)
	}

	_, err = tx.Exec(
		ctx,
		deleteUserQuery,
		userID,  // $1
		timeNow, // $2
	)
	if err != nil {
		return fmt.Errorf("tx.Exec deleteUserQuery: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}

	return nil
}
