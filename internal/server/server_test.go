package server

import (
	"bufio"
	stderrors "errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/errors"
	"github.com/harshithl1777/flock/internal/protocol"
)

type acceptResult struct {
	conn net.Conn
	err  error
}

type stubListener struct {
	acceptErr error
	accepts   chan acceptResult
	closeCh   chan struct{}
	closed    bool
	closeOnce sync.Once
}

func (l *stubListener) Accept() (net.Conn, error) {
	if l.accepts == nil {
		return nil, l.acceptErr
	}

	select {
	case <-l.closeCh:
		return nil, net.ErrClosed
	case result, ok := <-l.accepts:
		if !ok {
			return nil, net.ErrClosed
		}

		return result.conn, result.err
	}
}

func (l *stubListener) Close() error {
	l.closeOnce.Do(func() {
		if l.closeCh != nil {
			close(l.closeCh)
		}
	})

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

func TestServerStart_ConcurrentAccepts(t *testing.T) {
	ln := &stubListener{
		accepts: make(chan acceptResult, 2),
		closeCh: make(chan struct{}),
	}
	srv := New(newTestServerConfig())
	srv.listen = func(network, address string) (net.Listener, error) {
		return ln, nil
	}

	startErrCh := make(chan *errors.OpError, 1)
	go func() {
		startErrCh <- srv.Start()
	}()

	serverConn1, clientConn1 := net.Pipe()
	defer serverConn1.Close()
	defer clientConn1.Close()

	ln.accepts <- acceptResult{conn: serverConn1}

	serverConn2, clientConn2 := net.Pipe()
	defer serverConn2.Close()
	defer clientConn2.Close()

	ln.accepts <- acceptResult{conn: serverConn2}

	if err := clientConn2.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set second connection deadline: %v", err)
	}

	request := "" +
		"GET / HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"\r\n"
	if _, err := clientConn2.Write([]byte(request)); err != nil {
		t.Fatalf("write second request: %v", err)
	}

	response := readSingleResponse(t, clientConn2)
	if !strings.HasPrefix(response, "HTTP/1.1 200 OK\r\n") {
		t.Fatalf("second response missing status line: %q", response)
	}

	if !strings.Contains(response, `"status":"pass"`) {
		t.Fatalf("second response missing health payload: %q", response)
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	select {
	case err := <-startErrCh:
		if err != nil {
			t.Fatalf("got err %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Start to return")
	}

	if srv.ln != ln {
		t.Fatal("expected server to retain opened listener")
	}
}

func readSingleResponse(t *testing.T, conn net.Conn) string {
	t.Helper()

	reader := bufio.NewReader(conn)
	var response strings.Builder

	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read status line: %v", err)
	}
	response.WriteString(statusLine)

	contentLength := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read header line: %v", err)
		}

		response.WriteString(line)
		if line == "\r\n" {
			break
		}

		name, value, found := strings.Cut(strings.TrimRight(line, "\r\n"), ":")
		if !found {
			continue
		}

		if strings.EqualFold(strings.TrimSpace(name), string(protocol.HeaderContentLength)) {
			contentLength, err = strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				t.Fatalf("parse content length: %v", err)
			}
		}
	}

	if contentLength == 0 {
		return response.String()
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, body); err != nil {
		t.Fatalf("read response body: %v", err)
	}

	response.Write(body)
	return response.String()
}
