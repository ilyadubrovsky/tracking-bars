package user

import (
	"context"
)

type Repository interface {
	FindAll(ctx context.Context, aq ...string) ([]User, error)
	FindOne(ctx context.Context, id int64) (*User, error)
	UpdateProgressTable(ctx context.Context, id int64, table ProgressTable) error
	LogoutUser(ctx context.Context, id int64) error
}
