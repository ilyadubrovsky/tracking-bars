package service

import (
	"context"
	"errors"
	"github.com/ilyadubrovsky/bars"
	"time"
	"user-service/internal/entity/user"
	"user-service/pkg/logging"
)

var (
	ErrAlreadyAuthorized = errors.New("user already authorized")
	ErrNotAuthorized     = errors.New("user not authorized")
)

type userRepository interface {
	Create(ctx context.Context, u user.User) error
	GetAllUsers(ctx context.Context, aq ...string) ([]user.User, error)
	AuthorizationCheck(ctx context.Context, id int64) (*bool, error)
	Reauthorization(ctx context.Context, id int64, username string, password []byte, deleted bool) error
	LogoutUser(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
}

type Service struct {
	usersStorage userRepository
	logger       *logging.Logger
}

func (s *Service) Authorization(ctx context.Context, dto user.CreateUserDTO) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)

	defer cancel()

	isAuthorized, err := s.usersStorage.AuthorizationCheck(ctx, dto.ID)
	if err != nil {
		s.logger.Errorf("users repository: failed to AuthorizationCheck due to error: %v", err)
		return false, err
	}

	if isAuthorized != nil && *isAuthorized == false {
		return false, ErrAlreadyAuthorized
	}

	cl := bars.NewClient()

	err = cl.Authorization(ctx, dto.Username, dto.Password)

	if err != nil {
		if errors.Is(err, bars.ErrNoAuth) {
			return false, nil
		}
		s.logger.Errorf("failed to Authorization due to error: %v", err)
		s.logger.Debugf("UserID: %d, Username: %s", dto.ID, dto.Username)
		return false, err
	}

	usr := user.NewUser(dto)

	if err = usr.EncryptPassword(); err != nil {
		s.logger.Tracef("UserID: %d, username: %s", dto.ID, dto.Username)
		s.logger.Errorf("failed to encrypt a password due to error: %v", err)
		return false, err
	}

	if isAuthorized != nil && *isAuthorized == true {
		err = s.usersStorage.Reauthorization(ctx, usr.ID, usr.Username, usr.Password, usr.Deleted)
	} else {
		err = s.usersStorage.Create(ctx, usr)
	}

	if err != nil {
		s.logger.Errorf("users repository: failed to Create/Reauthorization due to error: %v", err)
		return false, err
	}

	return true, nil
}

func (s *Service) Logout(ctx context.Context, id int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)

	defer cancel()

	alreadyAuthorized, err := s.usersStorage.AuthorizationCheck(ctx, id)
	if err != nil {
		s.logger.Errorf("users repository: failed to AuthorizationCheck due to error: %v", err)
		return err
	}

	if alreadyAuthorized == nil || *alreadyAuthorized == true {
		return ErrNotAuthorized
	}

	if err = s.usersStorage.LogoutUser(ctx, id); err != nil {
		s.logger.Errorf("users repository: failed to LogoutUser due to error: %v", err)
		return err
	}

	return nil
}

func (s *Service) GetUsersByOpts(ctx context.Context, opts ...string) ([]user.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)

	defer cancel()

	usrs, err := s.usersStorage.GetAllUsers(ctx, opts...)
	if err != nil {
		s.logger.Errorf("users repository: failed to GetAllUsersID due to error: %v", err)
		return nil, err
	}

	return usrs, nil
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)

	defer cancel()

	if err := s.usersStorage.Delete(ctx, id); err != nil {
		s.logger.Errorf("users repository: failed to DeleteUser due to error: %v", err)
		return err
	}

	return nil
}

func NewService(usersStorage userRepository, logger *logging.Logger) *Service {
	return &Service{
		usersStorage: usersStorage,
		logger:       logger,
	}
}
