package user

import (
	"context"
	"fmt"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository"
)

type svc struct {
	usersRepository repository.Users
}

func NewService(
	usersRepository repository.Users,
) *svc {
	return &svc{
		usersRepository: usersRepository,
	}
}

func (s *svc) Save(ctx context.Context, user *domain.User) error {
	err := s.usersRepository.Save(ctx, user)
	if err != nil {
		return fmt.Errorf("usersRepository.Save: %w", err)
	}

	return nil
}

func (s *svc) Get(ctx context.Context, userID int64) (*domain.User, error) {
	user, err := s.usersRepository.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usersRepository.Get: %w", err)
	}

	return user, nil
}
