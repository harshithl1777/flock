package main

import (
	"github.com/harshithl1777/flock/core/config"
	"github.com/harshithl1777/flock/core/server"
	"github.com/harshithl1777/flock/core/utils/errors"
	"github.com/harshithl1777/flock/core/utils/logger"
)

const configPath = "config.yaml"

func readConfigYAML() *config.Config {
	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Fatal(errors.Wrap("load config", err))
	}
	return cfg
}

func main() {
	cfg := readConfigYAML()

	srv := server.New(cfg)
	err := srv.Start()
	logger.Info("starting server")

	if err != nil {
		logger.Fatal(errors.Wrap("server startup", err))
	}
}
