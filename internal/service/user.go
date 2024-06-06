package service

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type User interface {
	Save(ctx context.Context, user *domain.User) error
	User(ctx context.Context, userID int64) (*domain.User, error)
	// TODO добавить фильтр по deleted_at, фильтры. провалидировать всю логику
	Users(ctx context.Context) ([]*domain.User, error)
	Delete(ctx context.Context, userID int64) error
}
