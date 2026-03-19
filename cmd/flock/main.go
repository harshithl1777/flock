package main

import (
	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/errors"
	"github.com/harshithl1777/flock/internal/logger"
	"github.com/harshithl1777/flock/internal/server"
)

const configPath = ""

// readConfigYAML loads the server configuration.
//
// It terminates the process if the configuration cannot be loaded.
func readConfigYAML() *config.Config {
	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Fatal(errors.Wrap("load config", err))
	}
	return cfg
}

// main loads configuration, constructs the server, and starts serving requests.
func main() {
	cfg := readConfigYAML()

	srv := server.New(cfg)
	logger.Info("starting server")
	err := srv.Start()

	if err != nil {
		logger.Fatal(errors.Wrap("server startup", err))
	}
}
