package main

import (
	"log"

	"github.com/example/file-service/api"
	"github.com/example/file-service/config"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create server
	server, err := api.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	log.Printf("Starting file service on port %d with %s storage", cfg.Server.Port, cfg.Storage.Type)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}