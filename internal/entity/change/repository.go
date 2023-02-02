package change

import "context"

type Repository interface {
	Create(ctx context.Context, dto CreateChangeDTO) error
	FindAll(ctx context.Context) ([]Change, error)
	FindOne(ctx context.Context, id int) (Change, error)
	Update(ctx context.Context, dto UpdateChangeDTO) error
	Delete(ctx context.Context, id int) error
}
