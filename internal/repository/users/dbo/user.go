package dbo

import (
	"fmt"

	"github.com/ilyadubrovsky/tracking-bars/internal/domain"
	"github.com/jackc/pgtype"
)

func FromDomainToDBO(user *domain.User) (*pgtype.CompositeType, error) {
	vals := []interface{}{
		user.ID,
	}

	row := userTranscoder()
	if err := row.Set(vals); err != nil {
		return nil, fmt.Errorf("userTranscoder row.Set: %w", err)
	}

	return row, nil
}

var userTranscoder = func() *pgtype.CompositeType {
	fields := []pgtype.CompositeTypeField{
		{Name: "id"},
	}

	values := []pgtype.ValueTranscoder{
		&pgtype.Int8{},
	}

	rowType, _ := pgtype.NewCompositeTypeValues("user", fields, values)

	return rowType
}
