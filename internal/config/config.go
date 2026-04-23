package config

import (
	"bytes"
	_ "embed"
	"os"
	"time"

	"github.com/harshithl1777/flock/internal/errors"
	"github.com/harshithl1777/flock/internal/protocol"
	"gopkg.in/yaml.v3"
)

type NetworkConfig struct {
	Port                     int   `yaml:"port"`
	MaxRequestsPerConnection int64 `yaml:"maxRequestsPerConnection"`
}

type TimeoutsConfig struct {
	Read  time.Duration `yaml:"read"`
	Write time.Duration `yaml:"write"`
	Idle  time.Duration `yaml:"idle"`
}

type HandlerStatusOptions struct {
	Code protocol.StatusCode `yaml:"code"`
}

type HandlerRedirectOptions struct {
	Code        protocol.StatusCode `yaml:"code"`
	Destination string              `yaml:"destination"`
}

type HandlerHealthOptions struct {
	Code protocol.StatusCode `yaml:"code"`
}

type RouteConfig struct {
	StatusOptions   *HandlerStatusOptions   `yaml:"status,omitempty"`
	RedirectOptions *HandlerRedirectOptions `yaml:"redirect,omitempty"`
	HealthOptions   *HandlerHealthOptions   `yaml:"health,omitempty"`
	Path            string                  `yaml:"path"`
	Methods         []protocol.Method       `yaml:"methods"`
}

type RoutesConfig []RouteConfig

type Config struct {
	Network  NetworkConfig  `yaml:"network"`
	Routes   RoutesConfig   `yaml:"routes"`
	Timeouts TimeoutsConfig `yaml:"timeouts"`
}

func (nc NetworkConfig) Validate() *errors.OpError {
	if nc.Port < 1 || nc.Port > 65535 {
		return errors.Newf(errors.ConfigLoadKind, "validate network config", "invalid server port: %d", nc.Port)
	} else if nc.MaxRequestsPerConnection <= 0 {
		return errors.Newf(errors.ConfigLoadKind, "validate network config", "invalid maxRequestsPerConnection value: %d", nc.MaxRequestsPerConnection)
	}
	return nil
}

func (tc TimeoutsConfig) Validate() *errors.OpError {
	if tc.Read <= 0 {
		return errors.Newf(errors.ConfigLoadKind, "validate timeouts config", "invalid read timeout: %s", tc.Read)
	} else if tc.Write <= 0 {
		return errors.Newf(errors.ConfigLoadKind, "validate timeouts config", "invalid write timeout: %s", tc.Write)
	} else if tc.Idle <= 0 {
		return errors.Newf(errors.ConfigLoadKind, "validate timeouts config", "invalid idle timeout: %s", tc.Write)
	}

	return nil
}

func (o *HandlerHealthOptions) Validate() *errors.OpError {
	if !o.Code.IsValid() {
		return errors.Newf(errors.ConfigLoadKind, "validate health options", "invalid code: %d", o.Code)
	}
	return nil
}

func (o *HandlerStatusOptions) Validate() *errors.OpError {
	if !o.Code.IsValid() {
		return errors.Newf(errors.ConfigLoadKind, "validate status options", "invalid code: %d", o.Code)
	}
	return nil
}

func (o *HandlerRedirectOptions) Validate() *errors.OpError {
	if !o.Code.IsValid() {
		return errors.Newf(errors.ConfigLoadKind, "validate redirect options", "invalid code: %d", o.Code)
	}
	if o.Code < protocol.StatusMultipleChoices || o.Code >= protocol.StatusBadRequest {
		return errors.Newf(errors.ConfigLoadKind, "validate redirect options", "redirect code must be 3xx, received: %d", o.Code)
	}
	if o.Destination == "" || o.Destination[0] != '/' {
		return errors.Newf(errors.ConfigLoadKind, "validate redirect options", "invalid path: %s", o.Destination)
	}
	return nil
}

func (rc *RouteConfig) Validate() *errors.OpError {
	count := 0
	if rc.HealthOptions != nil {
		if err := rc.HealthOptions.Validate(); err != nil {
			return err
		}
		count++
	}
	if rc.StatusOptions != nil {
		if err := rc.StatusOptions.Validate(); err != nil {
			return err
		}
		count++
	}
	if rc.RedirectOptions != nil {
		if err := rc.RedirectOptions.Validate(); err != nil {
			return err
		}
		count++
	}

	if count != 1 {
		return errors.Newf(errors.ConfigLoadKind, "validate route config", "route %s: route received %d handler options, requires 1", rc.Path, count)
	}

	if rc.Path == "" || rc.Path[0] != '/' {
		return errors.Newf(errors.ConfigLoadKind, "validate route config", "invalid path: %s", rc.Path)
	}

	if len(rc.Methods) == 0 {
		return errors.Newf(errors.ConfigLoadKind, "validate route config", "route %s: at least one method required, received 0", rc.Path)
	}

	for _, method := range rc.Methods {
		if !method.IsValid() {
			return errors.Newf(errors.ConfigLoadKind, "validate route config", "route %s: invalid http method: %s", rc.Path, method)
		}
	}

	return nil
}

func (cfg *Config) Validate() *errors.OpError {
	if err := cfg.Network.Validate(); err != nil {
		return err
	}

	if err := cfg.Timeouts.Validate(); err != nil {
		return err
	}

	for _, rc := range cfg.Routes {
		if err := rc.Validate(); err != nil {
			return err
		}
	}

	return nil
}

//go:embed default.yaml
var defaultConfigBytes []byte

// Load reads, parses, and validates the YAML configuration file at path.
//
// It returns an *errors.OpError when the file cannot be read, the YAML is invalid, or
// the resulting configuration fails validation.
func Load(configFilePath string) (*Config, *errors.OpError) {
	var data []byte

	if configFilePath != "" {
		fdata, err := os.ReadFile(configFilePath)
		if err != nil {
			return nil, errors.Wrap(errors.ConfigLoadKind, "read file", err)
		}
		data = fdata
	} else {
		data = defaultConfigBytes
	}

	var cfg Config
	reader := bytes.NewReader(data)
	dec := yaml.NewDecoder(reader)
	dec.KnownFields(true)

	if err := dec.Decode(&cfg); err != nil {
		return nil, errors.Wrap(errors.ConfigLoadKind, "parse file", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
