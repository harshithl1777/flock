package config

import (
	_ "embed"
	"os"
	"time"

	"github.com/harshithl1777/flock/internal/errors"
	"gopkg.in/yaml.v3"
)

//go:embed default.yaml
var defaultConfigBytes []byte

type Config struct {
	Network struct {
		Port int `yaml:"port"`
	} `yaml:"network"`

	Timeouts struct {
		Read  time.Duration `yaml:"read"`
		Write time.Duration `yaml:"write"`
	} `yaml:"timeouts"`
}

// Load reads, parses, and validates the YAML configuration file at path.
//
// It returns an error when the file cannot be read, the YAML is invalid, or
// the resulting configuration fails validation.
func Load(configFilePath string) (*Config, error) {
	var data []byte
	var err error

	if configFilePath != "" {
		data, err = os.ReadFile(configFilePath)
		if err != nil {
			return nil, errors.Wrap("read file", err)
		}
	} else {
		data = defaultConfigBytes
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Wrap("parse file", err)
	}

	if cfg.Network.Port < 1 || cfg.Network.Port > 65535 {
		return nil, errors.Newf("validate config", "invalid server port: %d", cfg.Network.Port)
	}

	return &cfg, nil
}
