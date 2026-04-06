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
	routeHandler, errResponse := newTestRouter().Resolve(protocol.Get, "/")

	if errResponse != nil {
		t.Fatalf("got errResponse %v, want nil", errResponse)
	}

	if routeHandler == nil {
		t.Fatal("expected matched handler")
	}
}

func TestResolve_HeadAllowedByGet(t *testing.T) {
	routeHandler, errResponse := newTestRouter().Resolve(protocol.Head, "/")

	if errResponse != nil {
		t.Fatalf("got errResponse %v, want nil", errResponse)
	}

	if routeHandler == nil {
		t.Fatal("expected matched handler")
	}
}

func TestResolve_MethodNotAllowed(t *testing.T) {
	routeHandler, errResponse := newTestRouter().Resolve(protocol.Post, "/")

	if routeHandler != nil {
		t.Fatalf("got handler %v, want nil", routeHandler)
	}

	if errResponse == nil {
		t.Fatal("expected method not allowed errResponse")
	}

	if errResponse.StatusCode != int(protocol.StatusMethodNotAllowed) {
		t.Fatalf("got status %d, want %d", errResponse.StatusCode, protocol.StatusMethodNotAllowed)
	}

	if got := errResponse.Headers[protocol.HeaderAllow]; got != "GET, HEAD" {
		t.Fatalf("got allow header %q, want %q", got, "GET, HEAD")
	}
}

func TestResolve_OptionsReturnsOptionsDecision(t *testing.T) {
	routeHandler, errResponse := newTestRouter().Resolve(protocol.Options, "/")

	if routeHandler != nil {
		t.Fatalf("got handler %v, want nil", routeHandler)
	}

	if errResponse == nil {
		t.Fatal("expected options errResponse")
	}

	if errResponse.StatusCode != int(protocol.StatusNoContent) {
		t.Fatalf("got status %d, want %d", errResponse.StatusCode, protocol.StatusNoContent)
	}

	if got := errResponse.Headers[protocol.HeaderAllow]; got != "GET, HEAD" {
		t.Fatalf("got allow header %q, want %q", got, "GET, HEAD")
	}
}

func TestResolve_NotFound(t *testing.T) {
	routeHandler, errResponse := newTestRouter().Resolve(protocol.Get, "/missing")

	if routeHandler != nil {
		t.Fatalf("got handler %v, want nil", routeHandler)
	}

	if errResponse == nil {
		t.Fatal("expected not found errResponse")
	}

	if errResponse.StatusCode != int(protocol.StatusNotFound) {
		t.Fatalf("got status %d, want %d", errResponse.StatusCode, protocol.StatusNotFound)
	}
}
