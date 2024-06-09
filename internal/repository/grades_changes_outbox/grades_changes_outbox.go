package grades_changes_outbox

import (
	"context"
	"fmt"

	"github.com/ilyadubrovsky/tracking-bars/internal/database"
	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository/grades_changes_outbox/dbo"
)

type repo struct {
	db database.PG
}

func NewRepository(db database.PG) *repo {
	return &repo{db: db}
}

func (r *repo) GradesChanges(ctx context.Context, limit int64) ([]*domain.GradeChange, error) {
	query := `
		SELECT 
			id,
			user_id,
			grades_change,
			created_at
		FROM grades_changes_outbox
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("db.Query: %w", err)
	}
	defer rows.Close()

	gradesChanges := make([]*domain.GradeChange, 0)
	for rows.Next() {
		dboGradeChange := &dbo.GradeChange{}
		err = rows.Scan(
			&dboGradeChange.ID,
			&dboGradeChange.UserID,
			&dboGradeChange.Data,
			&dboGradeChange.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}

		gradeChange, err := dboGradeChange.ToDomain()
		if err != nil {
			return nil, fmt.Errorf("dboGradeChange.ToDomain: %w", err)
		}

		gradesChanges = append(gradesChanges, gradeChange)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}

	return gradesChanges, nil
}

func (r *repo) Delete(ctx context.Context, ids []int64) error {
	query := `
		DELETE FROM grades_changes_outbox
		WHERE id = ANY($1::BIGINT[])
	`

	_, err := r.db.Exec(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}
