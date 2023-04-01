package model

import (
	"context"
	"grades-service/internal/entity/user"
)

type Service interface {
	GetProgressTableFromDB(ctx context.Context, id int64) (*user.ProgressTable, error)
	UpdateAndGetProgressTable(ctx context.Context, id int64) (*user.ProgressTable, error)
}
