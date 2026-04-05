package router

import (
	"testing"

	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/protocol"
)

func newTestRouter() *Router {
	return New(config.RoutesConfig{
		{
			Path:    "/",
			Methods: []protocol.Method{protocol.Get},
			HealthOptions: &config.HandlerHealthOptions{
				Code: protocol.StatusOK,
			},
		},
		{
			Path:    "/users",
			Methods: []protocol.Method{protocol.Post},
			StatusOptions: &config.HandlerStatusOptions{
				Code: protocol.StatusCreated,
			},
		},
	})
}

func TestResolve_Dispatch(t *testing.T) {
	response, routeHandler := newTestRouter().Resolve(protocol.Get, "/")

	if response != nil {
		t.Fatalf("got response %v, want nil", response)
	}

	if routeHandler == nil {
		t.Fatal("expected matched handler")
	}
}

func TestResolve_HeadAllowedByGet(t *testing.T) {
	response, routeHandler := newTestRouter().Resolve(protocol.Head, "/")

	if response != nil {
		t.Fatalf("got response %v, want nil", response)
	}

	if routeHandler == nil {
		t.Fatal("expected matched handler")
	}
}

func TestResolve_MethodNotAllowed(t *testing.T) {
	response, routeHandler := newTestRouter().Resolve(protocol.Post, "/")

	if routeHandler != nil {
		t.Fatalf("got handler %v, want nil", routeHandler)
	}

	if response == nil {
		t.Fatal("expected method not allowed response")
	}

	if response.StatusCode != int(protocol.StatusMethodNotAllowed) {
		t.Fatalf("got status %d, want %d", response.StatusCode, protocol.StatusMethodNotAllowed)
	}

	if got := response.Headers[protocol.HeaderAllow]; got != "GET, HEAD" {
		t.Fatalf("got allow header %q, want %q", got, "GET, HEAD")
	}
}

func TestResolve_OptionsReturnsOptionsDecision(t *testing.T) {
	response, routeHandler := newTestRouter().Resolve(protocol.Options, "/")

	if routeHandler != nil {
		t.Fatalf("got handler %v, want nil", routeHandler)
	}

	if response == nil {
		t.Fatal("expected options response")
	}

	if response.StatusCode != int(protocol.StatusNoContent) {
		t.Fatalf("got status %d, want %d", response.StatusCode, protocol.StatusNoContent)
	}

	if got := response.Headers[protocol.HeaderAllow]; got != "GET, HEAD" {
		t.Fatalf("got allow header %q, want %q", got, "GET, HEAD")
	}
}

func TestResolve_NotFound(t *testing.T) {
	response, routeHandler := newTestRouter().Resolve(protocol.Get, "/missing")

	if routeHandler != nil {
		t.Fatalf("got handler %v, want nil", routeHandler)
	}

	if response == nil {
		t.Fatal("expected not found response")
	}

	if response.StatusCode != int(protocol.StatusNotFound) {
		t.Fatalf("got status %d, want %d", response.StatusCode, protocol.StatusNotFound)
	}
}
