package config

import (
	_ "embed"
	"os"
	"time"

	"github.com/harshithl1777/flock/internal/errors"
	"gopkg.in/yaml.v3"
)

type NetworkConfig struct {
	Port int `yaml:"port"`
}

type RouteConfig struct {
	Path    string   `yaml:"path"`
	Handler string   `yaml:"handler"`
	Methods []string `yaml:"methods"`
}

type RoutesConfig []RouteConfig

type TimeoutsConfig struct {
	Read  time.Duration `yaml:"read"`
	Write time.Duration `yaml:"write"`
}

type Config struct {
	Network  NetworkConfig  `yaml:"network"`
	Routes   RoutesConfig   `yaml:"routes"`
	Timeouts TimeoutsConfig `yaml:"timeouts"`
}

//go:embed default.yaml
var defaultConfigBytes []byte

// Load reads, parses, and validates the YAML configuration file at path.
//
// It returns an error when the file cannot be read, the YAML is invalid, or
// the resulting configuration fails validation.
func Load(configFilePath string) (*Config, *errors.OpError) {
	var data []byte
	var err error

	if configFilePath != "" {
		data, err = os.ReadFile(configFilePath)
		if err != nil {
			return nil, errors.Wrap(errors.ConfigLoadKind, "read file", err)
		}
	} else {
		data = defaultConfigBytes
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Wrap(errors.ConfigLoadKind, "parse file", err)
	}

	if cfg.Network.Port < 1 || cfg.Network.Port > 65535 {
		return nil, errors.Newf(errors.ConfigLoadKind, "validate config", "invalid server port: %d", cfg.Network.Port)
	}

	return &cfg, nil
}
