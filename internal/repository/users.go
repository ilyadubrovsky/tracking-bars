package repository

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type Users interface {
	Save(ctx context.Context, user *domain.User) error
	User(ctx context.Context, userID int64) (*domain.User, error)
	Users(ctx context.Context) ([]*domain.User, error)
	Delete(ctx context.Context, userID int64) error
	UpdateProgressTable(
		ctx context.Context,
		userID int64,
		progressTable *domain.ProgressTable,
		gradesChanges []*domain.GradeChange,
	) error
}
