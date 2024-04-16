package pg

import (
	"context"
	"fmt"

	"github.com/ilyadubrovsky/tracking-bars/internal/database"
	"github.com/jackc/pgx/v4/pgxpool"
)

func New(ctx context.Context, dsn string) (database.PG, error) {
	conn, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.Connect: %w", err)
	}

	if err = conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("conn.Ping: %w", err)
	}

	return conn, nil
}
