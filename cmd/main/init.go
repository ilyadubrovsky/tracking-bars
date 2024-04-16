package main

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

func initLogger(ctx context.Context) {
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}
