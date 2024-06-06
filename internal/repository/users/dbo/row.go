package dbo

import (
	"fmt"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type UserGetRow struct {
	ID            int64
	Username      *string
	Password      []byte
	ProgressTable []byte
}

func (d *UserGetRow) ToDomain() (*domain.User, error) {
	user := &domain.User{
		ID: d.ID,
	}

	if len(d.ProgressTable) != 0 {
		progressTable, err := ProgressTableToDomain(d.ProgressTable)
		if err != nil {
			return nil, fmt.Errorf("dbo.ProgressTableToDomain: %w", err)
		}
		user.ProgressTable = progressTable
	}

	if d.Username != nil && len(d.Password) != 0 {
		user.BarsCredentials = &domain.BarsCredentials{
			Username: *d.Username,
			Password: d.Password,
		}
	}

	return user, nil
}
