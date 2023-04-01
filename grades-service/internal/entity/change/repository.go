package change

import "context"

type Repository interface {
	Create(ctx context.Context, c Change) error
}
