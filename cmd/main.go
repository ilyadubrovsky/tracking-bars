package main

import (
	"log"

	"github.com/ilyadubrovsky/tracking-bars/internal/config"
)

func main() {
	_, err := config.NewConfig()
	if err != nil {
		log.Fatalf("cant initialize config: %v", err)
	}

	// TODO init postgresql

	// TODO init repository

	// TODO init services

	// TODO gracefully shutdown
}
