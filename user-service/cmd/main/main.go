package main

import (
	"github.com/joho/godotenv"
	"user-service/internal/app"
	"user-service/internal/config"
	"user-service/pkg/logging"
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

	if err = app.Run(cfg); err != nil {
		logger.Panic(err)
	}
}
