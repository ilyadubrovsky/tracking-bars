package pg

import (
	"context"
	"fmt"

	"github.com/ilyadubrovsky/tracking-bars/internal/database"
	"github.com/jackc/pgx/v5"
)

func New(ctx context.Context, dsn string) (database.PG, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgx.Connect: %w", err)
	}
	defer conn.Close(ctx)

	if err = conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("conn.Ping: %w", err)
	}

	return conn, nil
}
