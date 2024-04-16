package main

import (
	"context"
	"log"
	"time"

	"github.com/ilyadubrovsky/tracking-bars/internal/config"
	"github.com/ilyadubrovsky/tracking-bars/internal/database/pg"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository/bars_credentials"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository/progress_tables"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository/users"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/bars"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/grades_changes"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/progress_table"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/telegram"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/user"
	"github.com/rs/zerolog"
)

func main() {
	ctx := context.Background()

	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("cant initialize config: %v", err)
	}

	db, err := pg.New(ctx, cfg.Postgres.DSN)
	if err != nil {
		log.Fatalf("cant initialize postgresql: %v", err)
	}

	usersRepository := users.NewRepository(db)
	barsCredentialsRepository := bars_credentials.NewRepository(db)
	progressTablesRepository := progress_tables.NewRepository(db)

	userService := user.NewService(usersRepository)
	progressTableService := progress_table.NewService(progressTablesRepository)
	barsService := bars.NewService(
		progressTableService,
		userService,
		barsCredentialsRepository,
		cfg.Bars,
	)
	telegramService, err := telegram.NewService(
		userService,
		barsService,
		cfg.Telegram,
	)
	gradesChangesService := grades_changes.NewService(
		progressTableService,
		telegramService,
		barsService,
		barsCredentialsRepository,
		cfg.Bars,
	)
	if err != nil {
		log.Fatalf("cant initialize telegram service: %v", err)
	}

	gradesChangesService.Start()
	telegramService.Start()

	// TODO gracefully shutdown
}
