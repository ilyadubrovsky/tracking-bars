package service

import (
	"context"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository"
)

type svc struct {
	usersRepository repository.Users
}

func NewUser(
	usersRepository repository.Users,
) *svc {
	return &svc{
		usersRepository: usersRepository,
	}
}

func (s *svc) Save(ctx context.Context, user *domain.User) error {
	return nil
}
