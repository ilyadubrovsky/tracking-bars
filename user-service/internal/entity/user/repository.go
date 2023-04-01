package user

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, u User) error
	GetAllUsersID(ctx context.Context, aq ...string) ([]User, error)
	AuthorizationCheck(ctx context.Context, id int64) (*bool, error)
	Reauthorization(ctx context.Context, id int64, username string, password []byte, deleted bool) error
	LogoutUser(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
}
