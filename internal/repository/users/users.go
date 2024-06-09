package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ilyadubrovsky/tracking-bars/internal/database"
	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	dboOutbox "github.com/ilyadubrovsky/tracking-bars/internal/repository/grades_changes_outbox/dbo"
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
	AND bc.user_id IS NOT NULL
	AND bc.deleted_at IS NULL;
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

	deleteGradesChangesOutboxQuery := `
		DELETE FROM grades_changes_outbox
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
		deleteGradesChangesOutboxQuery,
		userID, // $1
	)
	if err != nil {
		return fmt.Errorf("tx.Exec deleteGradesChangesOutboxQuery: %w", err)
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

func (r *repo) UpdateProgressTable(
	ctx context.Context,
	userID int64,
	progressTable *domain.ProgressTable,
	gradesChanges []*domain.GradeChange,
) error {
	updateProgressTableQuery := `
		UPDATE progress_tables
		SET progress_table = $2, updated_at = $3
		WHERE user_id = $1
	`
	progressTableDBO, err := dbo.ProgressTableFromDomain(progressTable)
	if err != nil {
		return fmt.Errorf("dbo.ProgressTableFromDomain: %w", err)
	}

	timeNow := time.Now()
	outboxQuery, outboxValues, err := buildInsertGradesChangesOutboxQuery(gradesChanges, timeNow)
	if err != nil {
		return fmt.Errorf("buildInsertGradesChangesOutboxQuery: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db.Begin: %w", err)
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(
		ctx,
		updateProgressTableQuery,
		userID,           // $1
		progressTableDBO, // $2
		timeNow,          // $3
	)
	if err != nil {
		return fmt.Errorf("tx.Exec updateProgressTableQuery: %w", err)
	}

	if outboxQuery != "" && len(outboxValues) != 0 && result.RowsAffected() != 0 {
		_, err = tx.Exec(
			ctx,
			outboxQuery,
			outboxValues[0],
			outboxValues[1],
			outboxValues[2],
		)
		if err != nil {
			return fmt.Errorf("tx.Exec outboxQuery: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}

	return nil
}

func buildInsertGradesChangesOutboxQuery(gradesChanges []*domain.GradeChange, timeNow time.Time) (string, []interface{}, error) {
	if len(gradesChanges) == 0 {
		return "", nil, nil
	}

	query := `
		INSERT INTO grades_changes_outbox (user_id, grades_change, created_at)
		SELECT * FROM UNNEST($1::BIGINT[], $2::JSONB[], $3::TIMESTAMPTZ[])
	`

	userIDs := make([]int64, 0, len(gradesChanges))
	dboChanges := make([][]byte, 0, len(gradesChanges))
	createdAt := make([]time.Time, 0, len(gradesChanges))
	for _, gradeChange := range gradesChanges {
		userIDs = append(userIDs, gradeChange.UserID)
		gradeChangeDBO, err := dboOutbox.GradeChangeDataFromDomain(gradeChange)
		if err != nil {
			return "", nil, fmt.Errorf("dbo.GradeChangeFromDomain: %w", err)
		}
		dboChanges = append(dboChanges, gradeChangeDBO)
		createdAt = append(createdAt, timeNow)
	}

	return query, []interface{}{userIDs, dboChanges, createdAt}, nil
}
