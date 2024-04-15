package dbo

import (
	"time"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
)

type BarsCredentials struct {
	UserID    int64
	Username  string
	Password  []byte
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func (b *BarsCredentials) ToDomain() *domain.BarsCredentials {
	return &domain.BarsCredentials{
		UserID:   b.UserID,
		Username: b.Username,
		Password: b.Password,
	}
}
