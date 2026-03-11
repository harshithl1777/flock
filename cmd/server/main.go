package main

import (
	"log"

	"github.com/harshithl1777/flock/core/config"
	"github.com/harshithl1777/flock/core/server"
	"github.com/harshithl1777/flock/core/utils/errors"
)

const configPath = "config.yaml"

func main() {
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal(errors.Wrap("load config", err))
	}

	server := server.New(cfg)
	err = server.Start()
	// TODO: add custom logger
	log.Printf("Listening on port %d", cfg.Server.Port)

	if err != nil {
		log.Fatal(errors.Wrap("server startup", err))
	}
}
