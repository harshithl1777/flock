package server

import (
	"bytes"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/errors"
	"github.com/harshithl1777/flock/internal/http"
	"github.com/harshithl1777/flock/internal/logger"
	"github.com/harshithl1777/flock/internal/protocol"
	"github.com/harshithl1777/flock/internal/router"
)

type stubAddr string

func (a stubAddr) Network() string { return "tcp" }
func (a stubAddr) String() string  { return string(a) }

type recordingConn struct {
	bytes.Buffer
	writeLimit int
	writeErr   error
	closed     bool
}

func (c *recordingConn) Read(p []byte) (int, error)       { return 0, io.EOF }
func (c *recordingConn) Close() error                     { c.closed = true; return nil }
func (c *recordingConn) LocalAddr() net.Addr              { return stubAddr("local") }
func (c *recordingConn) RemoteAddr() net.Addr             { return stubAddr("remote") }
func (c *recordingConn) SetDeadline(time.Time) error      { return nil }
func (c *recordingConn) SetReadDeadline(time.Time) error  { return nil }
func (c *recordingConn) SetWriteDeadline(time.Time) error { return nil }
func (c *recordingConn) Write(p []byte) (int, error) {
	if c.writeErr == nil {
		return c.Buffer.Write(p)
	}

	limit := c.writeLimit
	if limit < 0 || limit > len(p) {
		limit = len(p)
	}

	if limit > 0 {
		if _, err := c.Buffer.Write(p[:limit]); err != nil {
			return 0, err
		}
	}

	return limit, c.writeErr
}

func testRouter() *router.Router {
	return router.New(config.RoutesConfig{
		{
			Path:    "/",
			Methods: []protocol.Method{protocol.Get},
			HealthOptions: &config.HandlerHealthOptions{
				Code: protocol.StatusOK,
			},
		},
	})
}

func serveRequest(t *testing.T, raw string) string {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	go NewConnection(serverConn, testRouter()).serve()

	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}

	request := "" +
		"GET / HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"\r\n"
	if raw == "" {
		raw = request
	}

	if _, err := clientConn.Write([]byte(raw)); err != nil {
		t.Fatalf("write request: %v", err)
	}

	responseBytes, err := io.ReadAll(clientConn)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	return string(responseBytes)
}

func newTestConnection(netConn net.Conn) *Connection {
	return &Connection{
		Conn:   netConn,
		router: testRouter(),
		log:    logger.With(logger.String("remote", "test")),
		remote: "test",
	}
}

func TestConnectionServe_WritesHTTPResponse(t *testing.T) {
	response := serveRequest(t, "")

	if !strings.HasPrefix(response, "HTTP/1.1 200 OK\r\n") {
		t.Fatalf("response missing status line: %q", response)
	}

	if !strings.Contains(response, "Content-Type: application/json; charset=utf-8\r\n") {
		t.Fatalf("response missing content type: %q", response)
	}

	if !strings.Contains(response, "Cache-Control: no-cache\r\n") {
		t.Fatalf("response missing cache-control header: %q", response)
	}

	if !strings.Contains(response, "Connection: close\r\n") {
		t.Fatalf("response missing connection header: %q", response)
	}

	if !strings.Contains(response, "Server: Flock/1.0\r\n") {
		t.Fatalf("response missing server header: %q", response)
	}

	if !strings.Contains(response, "X-Request-Id: r_") {
		t.Fatalf("response missing request id: %q", response)
	}

	if !strings.Contains(response, `"status":"pass"`) {
		t.Fatalf("response missing health payload: %q", response)
	}
}

func TestConnectionServe_MissingHostReturnsBadRequest(t *testing.T) {
	response := serveRequest(t, ""+
		"GET / HTTP/1.1\r\n"+
		"\r\n")

	if !strings.HasPrefix(response, "HTTP/1.1 400 Bad Request\r\n") {
		t.Fatalf("response missing bad request status line: %q", response)
	}

	if !strings.Contains(response, "Content-Type: application/json; charset=utf-8\r\n") {
		t.Fatalf("response missing json content type: %q", response)
	}

	if !strings.Contains(response, `{"error":"missing_host","description":"the Host header is required"}`) {
		t.Fatalf("response missing missing_host error body: %q", response)
	}
}

func TestConnectionServe_MethodNotAllowedIncludesAllowHeader(t *testing.T) {
	response := serveRequest(t, ""+
		"POST / HTTP/1.1\r\n"+
		"Host: localhost\r\n"+
		"\r\n")

	if !strings.HasPrefix(response, "HTTP/1.1 405 Method Not Allowed\r\n") {
		t.Fatalf("response missing method not allowed status line: %q", response)
	}

	if !strings.Contains(response, "Allow: GET, HEAD\r\n") {
		t.Fatalf("response missing allow header: %q", response)
	}
}

func TestConnectionServe_OptionsReturnsNoContentAndAllowHeader(t *testing.T) {
	response := serveRequest(t, ""+
		"OPTIONS / HTTP/1.1\r\n"+
		"Host: localhost\r\n"+
		"\r\n")

	if !strings.HasPrefix(response, "HTTP/1.1 204 No Content\r\n") {
		t.Fatalf("response missing no content status line: %q", response)
	}

	if !strings.Contains(response, "Allow: GET, HEAD\r\n") {
		t.Fatalf("response missing allow header: %q", response)
	}
}

func TestConnectionServe_NotFoundReturns404(t *testing.T) {
	response := serveRequest(t, ""+
		"GET /missing HTTP/1.1\r\n"+
		"Host: localhost\r\n"+
		"\r\n")

	if !strings.HasPrefix(response, "HTTP/1.1 404 Not Found\r\n") {
		t.Fatalf("response missing not found status line: %q", response)
	}

	if strings.Contains(response, "Allow:") {
		t.Fatalf("not found response should not include allow header: %q", response)
	}
}

func TestConnectionServe_PanicBeforeWriteReturnsInternalServerError(t *testing.T) {
	conn := &recordingConn{}
	c := newTestConnection(conn)

	c.serve()

	response := conn.String()
	if !strings.HasPrefix(response, "HTTP/1.1 500 Internal Server Error\r\n") {
		t.Fatalf("response missing internal server error status line: %q", response)
	}

	if !strings.Contains(response, "Connection: close\r\n") {
		t.Fatalf("response missing connection header: %q", response)
	}

	if !conn.closed {
		t.Fatal("expected connection to be closed")
	}
}

func TestConnectionSuccessWrite_WritesResponseAndTracksBytes(t *testing.T) {
	conn := &recordingConn{}
	c := newTestConnection(conn)
	ctx := newRequestContext()

	c.successWrite(ctx, http.NewStatusResponse(protocol.StatusAccepted))

	response := conn.String()
	if !strings.HasPrefix(response, "HTTP/1.1 202 Accepted\r\n") {
		t.Fatalf("response missing accepted status line: %q", response)
	}

	if !strings.Contains(response, "Server: Flock/1.0\r\n") {
		t.Fatalf("response missing server header: %q", response)
	}

	if !strings.Contains(response, "X-Request-Id: "+ctx.id+"\r\n") {
		t.Fatalf("response missing request id header: %q", response)
	}

	if c.bytesWritten == 0 {
		t.Fatal("expected bytesWritten to increase")
	}
}

func TestConnectionRouterWrite(t *testing.T) {
	testCases := []struct {
		name         string
		match        router.Match
		wantStatus   string
		wantAllow    string
		disallowText string
	}{
		{
			name: "options",
			match: router.Match{
				Route:    &router.Route{AllowHeader: "GET, HEAD"},
				Decision: router.Options,
			},
			wantStatus: "HTTP/1.1 204 No Content\r\n",
			wantAllow:  "Allow: GET, HEAD\r\n",
		},
		{
			name: "method not allowed",
			match: router.Match{
				Route:    &router.Route{AllowHeader: "GET, HEAD"},
				Decision: router.MethodNotAllowed,
				Err:      errors.New(errors.MethodNotAllowedKind, "match request route", "requested method POST not allowed"),
			},
			wantStatus: "HTTP/1.1 405 Method Not Allowed\r\n",
			wantAllow:  "Allow: GET, HEAD\r\n",
		},
		{
			name: "not found",
			match: router.Match{
				Decision: router.NotFound,
				Err:      errors.New(errors.NotFoundKind, "match request route", "no matching route found"),
			},
			wantStatus:   "HTTP/1.1 404 Not Found\r\n",
			disallowText: "Allow:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn := &recordingConn{}
			c := newTestConnection(conn)
			ctx := newRequestContext()

			c.routerWrite(ctx, tc.match)

			response := conn.String()
			if !strings.HasPrefix(response, tc.wantStatus) {
				t.Fatalf("response missing status line: %q", response)
			}

			if tc.wantAllow != "" && !strings.Contains(response, tc.wantAllow) {
				t.Fatalf("response missing allow header: %q", response)
			}

			if tc.disallowText != "" && strings.Contains(response, tc.disallowText) {
				t.Fatalf("response should not contain %q: %q", tc.disallowText, response)
			}
		})
	}
}

func TestConnectionFailWrite_WritesErrorResponse(t *testing.T) {
	conn := &recordingConn{}
	c := newTestConnection(conn)
	ctx := newRequestContext()

	c.failWrite(ctx, errors.New(errors.MissingHostKind, "parse headers", "missing host header"))

	response := conn.String()
	if !strings.HasPrefix(response, "HTTP/1.1 400 Bad Request\r\n") {
		t.Fatalf("response missing bad request status line: %q", response)
	}

	if !strings.Contains(response, `{"error":"missing_host","description":"the Host header is required"}`) {
		t.Fatalf("response missing missing_host body: %q", response)
	}
}

func TestConnectionPanicWrite_WritesInternalServerError(t *testing.T) {
	conn := &recordingConn{}
	c := newTestConnection(conn)
	ctx := newRequestContext()

	c.panicWrite(ctx)

	response := conn.String()
	if !strings.HasPrefix(response, "HTTP/1.1 500 Internal Server Error\r\n") {
		t.Fatalf("response missing internal server error status line: %q", response)
	}
}

func TestConnectionWrite_TracksPartialBytesOnWriteError(t *testing.T) {
	conn := &recordingConn{
		writeLimit: 16,
		writeErr:   io.ErrClosedPipe,
	}
	c := newTestConnection(conn)
	ctx := newRequestContext()

	c.write(ctx, http.NewTextResponse(protocol.StatusOK, "hello"), nil)

	if c.bytesWritten == 0 {
		t.Fatal("expected bytesWritten to record partial write progress")
	}

	if conn.Len() == 0 {
		t.Fatal("expected partial response bytes to be written")
	}
}
