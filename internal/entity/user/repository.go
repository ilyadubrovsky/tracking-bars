package user

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, dto CreateUserDTO) error
	FindAll(ctx context.Context, aq ...string) ([]User, error)
	FindOne(ctx context.Context, id int64) (User, error)
	Update(ctx context.Context, dto UpdateUserDTO) error
	Delete(ctx context.Context, id int64) error
}
