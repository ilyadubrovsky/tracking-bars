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

func ToDomain(dbo *BarsCredentials) *domain.BarsCredentials {
	return &domain.BarsCredentials{
		UserID:   dbo.UserID,
		Username: dbo.Username,
		Password: dbo.Password,
	}
}

func ManyToDomain(dbos []*BarsCredentials) []*domain.BarsCredentials {
	domains := make([]*domain.BarsCredentials, 0, len(dbos))
	for _, dbo := range dbos {
		domains = append(domains, ToDomain(dbo))
	}

	return domains
}
