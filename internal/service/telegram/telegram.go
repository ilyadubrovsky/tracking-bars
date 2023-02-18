package telegram

import (
	"context"
	"fmt"
	tele "gopkg.in/telebot.v3"
	"tracking-barsv1.1/internal/config"
	"tracking-barsv1.1/internal/entity/change"
	"tracking-barsv1.1/internal/entity/user"
	"tracking-barsv1.1/pkg/logging"
)

type Service struct {
	bot            *tele.Bot
	logger         *logging.Logger
	cfg            *config.Config
	usersStorage   user.Repository
	changesStorage change.Repository
}

func NewService(logger *logging.Logger, cfg *config.Config, usersStorage user.Repository, changesStorage change.Repository) *Service {
	return &Service{logger: logger, cfg: cfg, usersStorage: usersStorage, changesStorage: changesStorage}
}

func (s *Service) GetProgressTableByID(ctx context.Context, userID int64) (*user.ProgressTable, error) {
	usr, err := s.usersStorage.FindOne(ctx, userID)
	if err != nil {
		s.logger.Tracef("UserID: %d", userID)
		s.logger.Errorf("failed to FindOne due error: %v", err)
		return nil, err
	}

	if usr.Deleted == true || usr.ID != userID {
		return nil, nil
	}

	return &usr.ProgressTable, nil
}

func (s *Service) GetUserByID(ctx context.Context, userID int64) (*user.User, error) {
	usr, err := s.usersStorage.FindOne(ctx, userID)
	if err != nil {
		s.logger.Tracef("UserID: %d", userID)
		s.logger.Errorf("failed to FindOne due error: %v", err)
		return nil, err
	}

	if usr.Deleted == true || usr.ID != userID {
		return nil, nil
	}

	return &usr, nil
}

func (s *Service) LogoutUserByID(ctx context.Context, userID int64) error {
	usr, err := s.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Errorf("failed to LogoutUserByID method due error: %v", err)
		return err
	}
	usrDTO := user.UpdateUserDTO{
		ID:            userID,
		Username:      usr.Username,
		Password:      []byte{},
		ProgressTable: "{}",
		Deleted:       true,
	}

	if err = s.usersStorage.Update(ctx, usrDTO); err != nil {
		s.logger.Errorf("failed to Update due error: %v", err)
		return err
	}

	return nil
}

func (s *Service) DeleteUserByID(ctx context.Context, userID int64) error {
	if err := s.usersStorage.Delete(ctx, userID); err != nil {
		s.logger.Errorf("failed to Delete due error: %v", err)
		return err
	}

	return nil
}

func (s *Service) GetAllUsers(ctx context.Context, aq ...string) ([]user.User, error) {
	if len(aq) > 1 {
		return nil, fmt.Errorf("the length of aq is equal to %d, which is greater than one", len(aq))
	}

	users, err := s.usersStorage.FindAll(ctx, aq...)
	if err != nil {
		s.logger.Errorf("failed to FindAll due error: %s", err)
		return nil, err
	}

	return users, nil
}
