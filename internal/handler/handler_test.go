package handler

import (
	"strings"
	"testing"
	"time"

	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/http"
	"github.com/harshithl1777/flock/internal/protocol"
)

func TestHealthHandlerHandle_ReturnsJSONHealthPayload(t *testing.T) {
	previousStart := healthStartedAt
	healthStartedAt = time.Now().Add(-2*time.Hour - 45*time.Minute - 12*time.Second)
	defer func() {
		healthStartedAt = previousStart
	}()

	handler := NewHealthHandler(&config.HandlerHealthOptions{Code: protocol.StatusOK})

	response := handler.Handle(&http.Request{})

	if response.StatusCode != int(protocol.StatusOK) {
		t.Fatalf("got status %d, want %d", response.StatusCode, protocol.StatusOK)
	}

	if got := response.Headers[protocol.HeaderContentType]; got != "application/json; charset=utf-8" {
		t.Fatalf("got content type %q, want application/json; charset=utf-8", got)
	}

	if got := response.Headers[protocol.HeaderCacheControl]; got != "no-cache" {
		t.Fatalf("got cache-control %q, want no-cache", got)
	}

	for _, part := range []string{
		`"status":"pass"`,
		`"version":"1.0"`,
		`"uptime":"2h45m12s"`,
	} {
		if !strings.Contains(response.Body, part) {
			t.Fatalf("response body missing %q: %q", part, response.Body)
		}
	}
}

func TestStatusHandlerHandle_ReturnsConfiguredStatusCode(t *testing.T) {
	handler := NewStatusHandler(&config.HandlerStatusOptions{Code: protocol.StatusCreated})

	response := handler.Handle(&http.Request{})

	if response.StatusCode != int(protocol.StatusCreated) {
		t.Fatalf("got status %d, want %d", response.StatusCode, protocol.StatusCreated)
	}
}

func TestRedirectHandlerHandle_ReturnsConfiguredStatusAndLocation(t *testing.T) {
	handler := NewRedirectHandler(&config.HandlerRedirectOptions{
		Code:        protocol.StatusMovedPermanently,
		Destination: "/new-path",
	})

	response := handler.Handle(&http.Request{})

	if response.StatusCode != int(protocol.StatusMovedPermanently) {
		t.Fatalf("got status %d, want %d", response.StatusCode, protocol.StatusMovedPermanently)
	}

	if got := response.Headers[protocol.HeaderLocation]; got != "/new-path" {
		t.Fatalf("got location %q, want /new-path", got)
	}
}
