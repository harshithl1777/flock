package server

import (
	stderrors "errors"
	"net"
	"testing"
	"time"

	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/errors"
	"github.com/harshithl1777/flock/internal/protocol"
)

type stubListener struct {
	acceptErr error
	closed    bool
}

func (l *stubListener) Accept() (net.Conn, error) { return nil, l.acceptErr }
func (l *stubListener) Close() error {
	l.closed = true
	return nil
}
func (l *stubListener) Addr() net.Addr { return stubAddr("listener") }

func newTestServerConfig() *config.Config {
	return &config.Config{
		Network: config.NetworkConfig{Port: 8080},
		Timeouts: config.TimeoutsConfig{
			Read:  time.Second,
			Write: time.Second,
		},
		Routes: config.RoutesConfig{
			{
				Path:    "/",
				Methods: []protocol.Method{protocol.Get},
				HealthOptions: &config.HandlerHealthOptions{
					Code: protocol.StatusOK,
				},
			},
		},
	}
}

func TestNew_DefaultsListenFunc(t *testing.T) {
	srv := New(newTestServerConfig())

	if srv.listen == nil {
		t.Fatal("expected listen func to be initialized")
	}
}

func TestServerStart_ReturnsWrappedListenError(t *testing.T) {
	srv := New(newTestServerConfig())
	srv.listen = func(network, address string) (net.Listener, error) {
		if network != "tcp" {
			t.Fatalf("got network %q, want %q", network, "tcp")
		}
		if address != ":8080" {
			t.Fatalf("got address %q, want %q", address, ":8080")
		}
		return nil, stderrors.New("listen failed")
	}

	err := srv.Start()
	if err == nil {
		t.Fatal("expected startup error")
	}

	if err.Kind != errors.ServerStartupKind {
		t.Fatalf("got kind %v, want %v", err.Kind, errors.ServerStartupKind)
	}
}

func TestServerStart_ReturnsNilWhenListenerCloses(t *testing.T) {
	ln := &stubListener{acceptErr: net.ErrClosed}
	srv := New(newTestServerConfig())
	srv.listen = func(network, address string) (net.Listener, error) {
		return ln, nil
	}

	err := srv.Start()
	if err != nil {
		t.Fatalf("got err %v, want nil", err)
	}

	if srv.ln != ln {
		t.Fatal("expected server to retain opened listener")
	}
}
