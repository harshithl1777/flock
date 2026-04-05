package main

import (
	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/logger"
	"github.com/harshithl1777/flock/internal/server"
)

const configPath = "./flock.example.yaml"

// main loads configuration, constructs the server, and starts serving requests.
func main() {
	defer logger.Sync()

	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Fatal("failed to load config", logger.Err(err))
	}

	srv := server.New(cfg)
	logger.Info("starting server")
	err = srv.Start()

	if err != nil {
		logger.Fatal("failed to start server", logger.Err(err))
	}
}
