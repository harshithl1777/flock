package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	stderrors "github.com/harshithl1777/flock/internal/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewLogger_DevelopmentUsesConsoleEncoder(t *testing.T) {
	var buf bytes.Buffer

	log := newLogger("development", zapcore.AddSync(&buf), false)

	log.Info("request processed", zap.String("method", "GET"), zap.String("path", "/health"))

	got := buf.String()

	if strings.HasPrefix(strings.TrimSpace(got), "{") {
		t.Fatalf("expected console output, got JSON: %q", got)
	}

	if !strings.Contains(got, "INFO") {
		t.Fatalf("expected level in console output, got %q", got)
	}

	if !strings.Contains(got, "request processed") {
		t.Fatalf("expected message in console output, got %q", got)
	}

	if strings.Contains(got, "\t") {
		t.Fatalf("expected space-separated console columns, got tab-separated output %q", got)
	}

	if !strings.Contains(got, `{"method": "GET", "path": "/health"}`) {
		t.Fatalf("expected JSON fields in aligned final column, got %q", got)
	}
}

func TestNewLogger_ProductionUsesJSONEncoder(t *testing.T) {
	var buf bytes.Buffer

	log := newLogger("production", zapcore.AddSync(&buf), false)

	log.Info("request processed", zap.String("method", "GET"), zap.String("path", "/health"))

	got := strings.TrimSpace(buf.String())

	var decoded map[string]any
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("expected JSON log output, got %q: %v", got, err)
	}

	if decoded["msg"] != "request processed" {
		t.Fatalf("got message %v, want request processed", decoded["msg"])
	}

	if decoded["method"] != "GET" {
		t.Fatalf("got method %v, want GET", decoded["method"])
	}

	if decoded["path"] != "/health" {
		t.Fatalf("got path %v, want /health", decoded["path"])
	}
}

func TestNewLogger_EmptyEnvDefaultsToJSONEncoder(t *testing.T) {
	var buf bytes.Buffer

	log := newLogger("", zapcore.AddSync(&buf), false)

	log.Info("request processed", zap.String("method", "GET"))

	got := strings.TrimSpace(buf.String())

	var decoded map[string]any
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("expected JSON log output for empty env, got %q: %v", got, err)
	}

	if decoded["msg"] != "request processed" {
		t.Fatalf("got message %v, want request processed", decoded["msg"])
	}

	if decoded["method"] != "GET" {
		t.Fatalf("got method %v, want GET", decoded["method"])
	}
}

func TestErr_OpErrorUsesStructuredObject(t *testing.T) {
	var buf bytes.Buffer

	log := newLogger("production", zapcore.AddSync(&buf), false)

	log.Error("request failed", Err(stderrors.Newf(stderrors.MalformedRequestLineKind, "read request", "invalid http method: %s", "TRACE")))

	got := strings.TrimSpace(buf.String())

	var decoded map[string]any
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("expected JSON log output, got %q: %v", got, err)
	}

	errorField, ok := decoded["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected structured error object, got %T (%v)", decoded["error"], decoded["error"])
	}

	if errorField["op"] != "read request" {
		t.Fatalf("got op %v, want read request", errorField["op"])
	}

	if errorField["error"] != "invalid http method: TRACE" {
		t.Fatalf("got nested error %v, want full message", errorField["error"])
	}
}

func TestNewLogger_DevelopmentAlignsJSONContextColumn(t *testing.T) {
	var buf bytes.Buffer

	log := newLogger("development", zapcore.AddSync(&buf), false)

	log.Info("starting server")
	log.Error("test error", zap.String("who", "me"))
	log.Info("listening", zap.String("addr", ":8080"))

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 log lines, got %d: %q", len(lines), buf.String())
	}

	errorIndex := strings.Index(lines[1], `{"who": "me"}`)
	listeningIndex := strings.Index(lines[2], `{"addr": ":8080"}`)

	if errorIndex == -1 || listeningIndex == -1 {
		t.Fatalf("expected JSON context in aligned lines, got %q", buf.String())
	}

	if errorIndex != listeningIndex {
		t.Fatalf("expected aligned JSON context column, got indexes %d and %d in %q", errorIndex, listeningIndex, buf.String())
	}
}
