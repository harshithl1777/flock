package config

import (
	"fmt"
	"os"
	"time"

	"github.com/harshithl1777/flock/core/utils/errors"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Network struct {
		Port int `yaml:"port"`
	} `yaml:"network"`

	Timeouts struct {
		Read  time.Duration `yaml:"read"`
		Write time.Duration `yaml:"write"`
	} `yaml:"timeouts"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap("read file", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, errors.Wrap("parse file", err)
	}

	if config.Network.Port == 0 {
		return nil, errors.Wrap("validate config", fmt.Errorf("server port is undefined"))
	}

	return &config, nil
}
