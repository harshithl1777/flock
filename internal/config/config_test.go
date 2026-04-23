package config

import (
	stderrors "errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harshithl1777/flock/internal/errors"
	"github.com/harshithl1777/flock/internal/protocol"
)

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `
network:
  port: 8080
  maxRequestsPerConnection: 100
routes:
  - path: /
    methods: [GET]
    health:
      code: 200
  - path: /users
    methods: [GET, POST]
    status:
      code: 201
timeouts:
  read: 5s
  write: 10s
  idle: 10s
`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Network.Port != 8080 {
		t.Fatalf("got port %d, want 8080", cfg.Network.Port)
	}

	if len(cfg.Routes) != 2 {
		t.Fatalf("got %d routes, want 2", len(cfg.Routes))
	}

	if cfg.Routes[0].Path != "/" {
		t.Fatalf("got first route path %q, want /", cfg.Routes[0].Path)
	}

	if cfg.Routes[0].HealthOptions == nil {
		t.Fatal("expected first route health options")
	}

	if cfg.Routes[0].HealthOptions.Code != protocol.StatusOK {
		t.Fatalf("got first route health code %d, want %d", cfg.Routes[0].HealthOptions.Code, protocol.StatusOK)
	}

	if len(cfg.Routes[0].Methods) != 1 || cfg.Routes[0].Methods[0] != protocol.Get {
		t.Fatalf("got first route methods %v, want [GET]", cfg.Routes[0].Methods)
	}

	if cfg.Routes[1].Path != "/users" {
		t.Fatalf("got second route path %q, want /users", cfg.Routes[1].Path)
	}

	if cfg.Routes[1].StatusOptions == nil {
		t.Fatal("expected second route status options")
	}

	if cfg.Routes[1].StatusOptions.Code != protocol.StatusCreated {
		t.Fatalf("got second route status code %d, want %d", cfg.Routes[1].StatusOptions.Code, protocol.StatusCreated)
	}

	if len(cfg.Routes[1].Methods) != 2 || cfg.Routes[1].Methods[0] != protocol.Get || cfg.Routes[1].Methods[1] != protocol.Post {
		t.Fatalf("got second route methods %v, want [GET POST]", cfg.Routes[1].Methods)
	}

	if cfg.Timeouts.Read != 5*time.Second {
		t.Fatalf("got read timeout %v, want 5s", cfg.Timeouts.Read)
	}

	if cfg.Timeouts.Write != 10*time.Second {
		t.Fatalf("got write timeout %v, want 10s", cfg.Timeouts.Write)
	}

	if cfg.Timeouts.Idle != 10*time.Second {
		t.Fatalf("got idle timeout %v, want 10s", cfg.Timeouts.Idle)
	}

	if cfg.Network.MaxRequestsPerConnection != 100 {
		t.Fatalf("got max requests per connection %d, want 100", cfg.Network.MaxRequestsPerConnection)
	}
}

func TestLoad_DefaultConfigWhenPathEmpty(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Network.Port != 8080 {
		t.Fatalf("got port %d, want 8080", cfg.Network.Port)
	}

	if cfg.Network.MaxRequestsPerConnection != 100 {
		t.Fatalf("got max requests per connection %d, want 100", cfg.Network.MaxRequestsPerConnection)
	}

	if len(cfg.Routes) != 1 {
		t.Fatalf("got %d routes, want 1", len(cfg.Routes))
	}

	if cfg.Routes[0].Path != "/health" {
		t.Fatalf("got route path %q, want /health", cfg.Routes[0].Path)
	}

	if cfg.Routes[0].HealthOptions == nil {
		t.Fatal("expected default route health options")
	}

	if cfg.Routes[0].HealthOptions.Code != protocol.StatusOK {
		t.Fatalf("got route health code %d, want %d", cfg.Routes[0].HealthOptions.Code, protocol.StatusOK)
	}

	if len(cfg.Routes[0].Methods) != 1 || cfg.Routes[0].Methods[0] != protocol.Get {
		t.Fatalf("got route methods %v, want [GET]", cfg.Routes[0].Methods)
	}

	if cfg.Timeouts.Read != 5*time.Second {
		t.Fatalf("got read timeout %v, want 5s", cfg.Timeouts.Read)
	}

	if cfg.Timeouts.Write != 10*time.Second {
		t.Fatalf("got write timeout %v, want 10s", cfg.Timeouts.Write)
	}

	if cfg.Timeouts.Idle != 10*time.Second {
		t.Fatalf("got idle timeout %v, want 10s", cfg.Timeouts.Idle)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("does-not-exist.yaml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var opErr *errors.OpError
	if !stderrors.As(err, &opErr) {
		t.Fatalf("expected *errors.OpError, got %T", err)
	}
}

func TestLoad_MissingPort(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `
network:
  maxRequestsPerConnection: 100
timeouts:
  read: 5s
  write: 10s
  idle: 10s
`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var opErr *errors.OpError
	if !stderrors.As(err, &opErr) {
		t.Fatalf("expected *errors.OpError, got %T", err)
	}
}
