package main

import (
	"github.com/joho/godotenv"
	"tracking-barsv1.1/internal/app"
	"tracking-barsv1.1/internal/config"
	"tracking-barsv1.1/pkg/logging"
)

func main() {
	logger := logging.GetLogger()

	if err := godotenv.Load(); err != nil {
		logger.Panic(err)
	}

	cfg, err := config.GetConfig()
	if err != nil {
		logger.Panic(err)
	}

	a, err := app.NewApp(cfg)
	if err != nil {
		logger.Panic(err)
	}

	if err = a.Run(); err != nil {
		logger.Panic(err)
	}
}
