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

func (s *svc) User(ctx context.Context, userID int64) (*domain.User, error) {
	user, err := s.usersRepository.User(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("usersRepository.User: %w", err)
	}

	return user, nil
}

func (s *svc) Users(ctx context.Context) ([]*domain.User, error) {
	users, err := s.usersRepository.Users(ctx)
	if err != nil {
		return nil, fmt.Errorf("usersRepository.Users: %w", err)
	}

	return users, nil
}

func (s *svc) Delete(ctx context.Context, userID int64) error {
	err := s.usersRepository.Delete(ctx, userID)
	if err != nil {
		return fmt.Errorf("usersRepository.Delete: %w", err)
	}

	return nil
}

func (s *svc) UpdateProgressTable(
	ctx context.Context,
	userID int64,
	progressTable *domain.ProgressTable,
	gradesChanges []*domain.GradeChange,
) error {
	err := s.usersRepository.UpdateProgressTable(ctx, userID, progressTable, gradesChanges)
	if err != nil {
		return fmt.Errorf("usersRepository.UpdateProgressTable: %w", err)
	}

	return nil
}
