package main

import (
	"context"
	"log"
	"time"

	"github.com/ilyadubrovsky/tracking-bars/internal/config"
	"github.com/ilyadubrovsky/tracking-bars/internal/database/pg"
	gradeschangesoutboxrepo "github.com/ilyadubrovsky/tracking-bars/internal/repository/grades_changes_outbox"
	"github.com/ilyadubrovsky/tracking-bars/internal/repository/users"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/bars"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/grades_changes"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/grades_changes_outbox"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/telegram"
	"github.com/ilyadubrovsky/tracking-bars/internal/service/user"
	"github.com/jellydator/ttlcache/v3"
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
	gradesChangesOutboxRepository := gradeschangesoutboxrepo.NewRepository(db)
	authorizationFailedRetriesCountCache := ttlcache.New[int64, int](
		ttlcache.WithTTL[int64, int](60 * time.Minute),
	)

	userService := user.NewService(usersRepository)
	barsService := bars.NewService(
		userService,
		cfg.Bars,
	)
	telegramService, err := telegram.NewService(
		userService,
		barsService,
		cfg.Telegram,
	)
	if err != nil {
		log.Fatalf("cant initialize telegram service: %v", err)
	}
	gradesChangesService := grades_changes.NewService(
		telegramService,
		barsService,
		userService,
		authorizationFailedRetriesCountCache,
		cfg.Bars,
	)
	gradesChangesOutboxService := grades_changes_outbox.NewService(
		gradesChangesOutboxRepository,
		telegramService,
		cfg.Bars,
	)

	go gradesChangesOutboxService.Start()
	go authorizationFailedRetriesCountCache.Start()
	go gradesChangesService.Start()
	telegramService.Start()

	// TODO gracefully shutdown
}
