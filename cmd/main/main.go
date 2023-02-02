package main

import (
	"TrackingBARSv2/internal/app"
	"TrackingBARSv2/internal/config"
	"TrackingBARSv2/pkg/logging"
	"github.com/joho/godotenv"
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
