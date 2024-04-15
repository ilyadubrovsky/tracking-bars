package main

import (
	"context"
	"log"

	"github.com/ilyadubrovsky/tracking-bars/internal/config"
	"github.com/ilyadubrovsky/tracking-bars/internal/database/pg"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository/bars_credentials"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository/progress_tables"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository/users"
)

func main() {
	ctx := context.Background()

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("cant initialize config: %v", err)
	}

	db, err := pg.New(ctx, cfg.Postgres.DSN)
	if err != nil {
		log.Fatalf("cant initialize postgresql: %v", err)
	}

	bars_credentials.NewRepository(db)
	users.NewRepository(db)
	progress_tables.NewRepository(db)

	// TODO init services

	// TODO gracefully shutdown
}
