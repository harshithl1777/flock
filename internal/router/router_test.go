package router

import (
	"testing"

	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/errors"
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
	match := newTestRouter().Resolve(protocol.Get, "/")

	if match.Err != nil {
		t.Fatalf("got err %v, want nil", match.Err)
	}

	if match.Decision != Forward {
		t.Fatalf("got decision %v, want %v", match.Decision, Forward)
	}

	if match.Route == nil {
		t.Fatal("expected matched handler")
	}
}

func TestResolve_HeadAllowedByGet(t *testing.T) {
	match := newTestRouter().Resolve(protocol.Head, "/")

	if match.Err != nil {
		t.Fatalf("got err %v, want nil", match.Err)
	}

	if match.Decision != Forward {
		t.Fatalf("got decision %v, want %v", match.Decision, Forward)
	}

	if match.Route == nil {
		t.Fatal("expected matched handler")
	}
}

func TestResolve_MethodNotAllowed(t *testing.T) {
	match := newTestRouter().Resolve(protocol.Post, "/")

	if match.Route == nil {
		t.Fatal("expected matched route for allow header")
	}

	if match.Err == nil {
		t.Fatal("expected method not allowed err")
	}

	if match.Decision != MethodNotAllowed {
		t.Fatalf("got decision %v, want %v", match.Decision, MethodNotAllowed)
	}

	if match.Err.Kind != errors.MethodNotAllowedKind {
		t.Fatalf("got error kind %v, want %v", match.Err.Kind, errors.MethodNotAllowedKind)
	}

	if match.Route.AllowHeader != "GET, HEAD" {
		t.Fatalf("got allow header %q, want %q", match.Route.AllowHeader, "GET, HEAD")
	}
}

func TestResolve_OptionsUsesPathMatch(t *testing.T) {
	match := newTestRouter().Resolve(protocol.Options, "/")

	if match.Route == nil {
		t.Fatal("expected options route match")
	}

	if match.Err != nil {
		t.Fatalf("got err %v, want nil", match.Err)
	}

	if match.Decision != Options {
		t.Fatalf("got decision %v, want %v", match.Decision, Options)
	}

	if match.Route.AllowHeader != "GET, HEAD" {
		t.Fatalf("got allow header %q, want %q", match.Route.AllowHeader, "GET, HEAD")
	}
}

func TestResolve_NotFound(t *testing.T) {
	match := newTestRouter().Resolve(protocol.Get, "/missing")

	if match.Route != nil {
		t.Fatalf("got route %v, want nil", match.Route)
	}

	if match.Err == nil {
		t.Fatal("expected not found err")
	}

	if match.Decision != NotFound {
		t.Fatalf("got decision %v, want %v", match.Decision, NotFound)
	}

	if match.Err.Kind != errors.NotFoundKind {
		t.Fatalf("got error kind %v, want %v", match.Err.Kind, errors.NotFoundKind)
	}
}
