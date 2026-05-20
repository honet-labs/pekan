package main

import (
	"log"

	"pekan/backend/internal/app"
	"pekan/backend/internal/platform/config"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	server, err := app.NewServer(cfg)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	if err := server.Run(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
