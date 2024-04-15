package main

import (
	"context"
	"log"

	"github.com/ilyadubrovsky/tracking-bars/internal/config"
	"github.com/ilyadubrovsky/tracking-bars/internal/database/pg"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository/bars_credentials"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository/progress_tables"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository/users"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/bars_credential"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/grades_changes"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/progress_table"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/telegram"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/user"
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

	usersRepository := users.NewRepository(db)
	barsCredentialsRepository := bars_credentials.NewRepository(db)
	progressTablesRepository := progress_tables.NewRepository(db)

	userService := user.NewService(usersRepository)
	barsCredentialService := bars_credential.NewService(
		userService,
		barsCredentialsRepository,
		cfg.Bars,
	)
	progressTableService := progress_table.NewService(
		barsCredentialService,
		progressTablesRepository,
	)
	telegramService, err := telegram.NewService(
		userService,
		barsCredentialService,
		cfg.Telegram,
	)
	gradesChangesService := grades_changes.NewService(
		progressTableService,
		barsCredentialService,
		telegramService,
		cfg.Bars,
	)
	if err != nil {
		log.Fatalf("cant initialize telegram service: %v", err)
	}

	gradesChangesService.Start()
	telegramService.Start()

	// TODO gracefully shutdown
}
