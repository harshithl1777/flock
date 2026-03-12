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
