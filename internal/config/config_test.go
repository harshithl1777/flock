package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `
network:
  port: 8080
routes:
  - path: /
    handler: health
    methods: [GET]
  - path: /users
    handler: users
    methods: [GET, POST]
timeouts:
  read: 5s
  write: 10s
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

	if cfg.Routes[0].Handler != "health" {
		t.Fatalf("got first route handler %q, want health", cfg.Routes[0].Handler)
	}

	if len(cfg.Routes[0].Methods) != 1 || cfg.Routes[0].Methods[0] != "GET" {
		t.Fatalf("got first route methods %v, want [GET]", cfg.Routes[0].Methods)
	}

	if cfg.Routes[1].Path != "/users" {
		t.Fatalf("got second route path %q, want /users", cfg.Routes[1].Path)
	}

	if cfg.Routes[1].Handler != "users" {
		t.Fatalf("got second route handler %q, want users", cfg.Routes[1].Handler)
	}

	if len(cfg.Routes[1].Methods) != 2 || cfg.Routes[1].Methods[0] != "GET" || cfg.Routes[1].Methods[1] != "POST" {
		t.Fatalf("got second route methods %v, want [GET POST]", cfg.Routes[1].Methods)
	}

	if cfg.Timeouts.Read != 5*time.Second {
		t.Fatalf("got read timeout %v, want 5s", cfg.Timeouts.Read)
	}

	if cfg.Timeouts.Write != 10*time.Second {
		t.Fatalf("got write timeout %v, want 10s", cfg.Timeouts.Write)
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

	if len(cfg.Routes) != 1 {
		t.Fatalf("got %d routes, want 1", len(cfg.Routes))
	}

	if cfg.Routes[0].Path != "/" {
		t.Fatalf("got route path %q, want /", cfg.Routes[0].Path)
	}

	if cfg.Routes[0].Handler != "health" {
		t.Fatalf("got route handler %q, want health", cfg.Routes[0].Handler)
	}

	if len(cfg.Routes[0].Methods) != 1 || cfg.Routes[0].Methods[0] != "GET" {
		t.Fatalf("got route methods %v, want [GET]", cfg.Routes[0].Methods)
	}

	if cfg.Timeouts.Read != 5*time.Second {
		t.Fatalf("got read timeout %v, want 5s", cfg.Timeouts.Read)
	}

	if cfg.Timeouts.Write != 10*time.Second {
		t.Fatalf("got write timeout %v, want 10s", cfg.Timeouts.Write)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("does-not-exist.yaml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoad_MissingPort(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `
timeouts:
  read: 5s
  write: 10s
`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
