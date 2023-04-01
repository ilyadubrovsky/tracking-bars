package model

import (
	"context"
	"user-service/internal/entity/user"
)

type Service interface {
	Authorization(ctx context.Context, dto user.CreateUserDTO) (bool, error)
	Logout(ctx context.Context, id int64) error
	GetUsersIDByOpts(ctx context.Context, opts ...string) ([]user.User, error)
	DeleteUser(ctx context.Context, id int64) error
}
