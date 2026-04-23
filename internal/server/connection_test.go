package server

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"strconv"
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

func testConfig() *config.Config {
	return &config.Config{
		Network: config.NetworkConfig{
			MaxRequestsPerConnection: 100,
		},
		Timeouts: config.TimeoutsConfig{
			Read:  50 * time.Millisecond,
			Write: time.Second,
			Idle:  time.Second,
		},
	}
}

func serveRequest(t *testing.T, raw string) string {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	go NewConnection(serverConn, testConfig(), testRouter()).serve()

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

	return readHTTPResponse(t, clientConn)
}

func readHTTPResponse(t *testing.T, conn net.Conn) string {
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

func newTestConnection(netConn net.Conn) *Connection {
	return &Connection{
		Conn:   netConn,
		cfg:    testConfig(),
		router: testRouter(),
		ctx:    newConnectionContext(),
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

	if !strings.Contains(response, "Server: Flock/1.0\r\n") {
		t.Fatalf("response missing server header: %q", response)
	}

	if !strings.Contains(response, "X-Request-Id: ") {
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

func TestConnectionServe_ReadTimeoutReturnsRequestTimeout(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	go NewConnection(serverConn, testConfig(), testRouter()).serve()

	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}

	response := readHTTPResponse(t, clientConn)

	if !strings.HasPrefix(response, "HTTP/1.1 408 Request Timeout\r\n") {
		t.Fatalf("response missing request timeout status line: %q", response)
	}

	if !strings.Contains(response, `{"error":"request_timeout","description":"the client took too long to send the request"}`) {
		t.Fatalf("response missing request timeout body: %q", response)
	}
}

func TestConnectionServe_KeepAliveIdleTimeoutClosesQuietly(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	cfg := testConfig()
	cfg.Timeouts.Idle = 50 * time.Millisecond

	go NewConnection(serverConn, cfg, testRouter()).serve()

	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}

	request := "" +
		"GET / HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"\r\n"
	if _, err := clientConn.Write([]byte(request)); err != nil {
		t.Fatalf("write request: %v", err)
	}

	response := readHTTPResponse(t, clientConn)
	if !strings.HasPrefix(response, "HTTP/1.1 200 OK\r\n") {
		t.Fatalf("response missing status line: %q", response)
	}

	buf := make([]byte, 1)
	n, err := clientConn.Read(buf)
	if err != io.EOF {
		t.Fatalf("expected idle timeout to close connection, got n=%d err=%v", n, err)
	}
	if n != 0 {
		t.Fatalf("expected no additional bytes after idle timeout, got %d", n)
	}
}

func TestConnectionServe_MaxRequestsPerConnectionClosesAfterLimit(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	cfg := testConfig()
	cfg.Network.MaxRequestsPerConnection = 1

	go NewConnection(serverConn, cfg, testRouter()).serve()

	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}

	request := "" +
		"GET / HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"\r\n"
	if _, err := clientConn.Write([]byte(request)); err != nil {
		t.Fatalf("write request: %v", err)
	}

	response := readHTTPResponse(t, clientConn)
	if !strings.HasPrefix(response, "HTTP/1.1 200 OK\r\n") {
		t.Fatalf("response missing status line: %q", response)
	}

	if !strings.Contains(response, "Connection: close\r\n") {
		t.Fatalf("expected close header on last allowed response: %q", response)
	}

	buf := make([]byte, 1)
	n, err := clientConn.Read(buf)
	if err != io.EOF {
		t.Fatalf("expected connection close after max requests, got n=%d err=%v", n, err)
	}
	if n != 0 {
		t.Fatalf("expected no additional bytes after max request close, got %d", n)
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

func TestConnectionInit_SetsReaderWhenNil(t *testing.T) {
	conn := &recordingConn{}
	c := newTestConnection(conn)

	if c.reader != nil {
		t.Fatal("expected test connection reader to start nil")
	}

	c.init()

	if c.reader == nil {
		t.Fatal("expected init to initialize reader")
	}
}

func TestConnectionInit_PreservesExistingReader(t *testing.T) {
	conn := &recordingConn{}
	c := newTestConnection(conn)
	existing := bufio.NewReader(strings.NewReader("GET / HTTP/1.1\r\n"))
	c.reader = existing

	c.init()

	if c.reader != existing {
		t.Fatal("expected init to preserve existing reader")
	}
}

func TestConnectionServe_InitializesReaderForManuallyConstructedConnection(t *testing.T) {
	conn := &recordingConn{}
	c := newTestConnection(conn)
	c.serve()

	if c.reader == nil {
		t.Fatal("expected serve to initialize reader")
	}

	if !conn.closed {
		t.Fatal("expected connection to be closed")
	}
}

func TestConnectionHandle_ClientCloseDoesNotWriteErrorResponse(t *testing.T) {
	conn := &recordingConn{}
	c := newTestConnection(conn)
	c.reader = bufio.NewReader(strings.NewReader(""))

	close := c.handle()

	if !close {
		t.Fatal("expected handle to close after client disconnect")
	}

	if conn.Len() != 0 {
		t.Fatalf("expected no response to be written, got %q", conn.String())
	}
}

func TestConnectionServe_ClientCloseBeforeRequestWritesNothing(t *testing.T) {
	conn := &recordingConn{}
	c := newTestConnection(conn)

	c.serve()

	response := conn.String()
	if response != "" {
		t.Fatalf("expected no response for clean client close, got %q", response)
	}

	if !conn.closed {
		t.Fatal("expected connection to be closed")
	}
}

func TestConnectionSuccessWrite_WritesResponseAndTracksBytes(t *testing.T) {
	conn := &recordingConn{}
	c := newTestConnection(conn)
	ctx := c.newRequestContext()

	c.successWrite(ctx, http.NewStatusResponse(protocol.StatusAccepted))

	response := conn.String()
	if !strings.HasPrefix(response, "HTTP/1.1 202 Accepted\r\n") {
		t.Fatalf("response missing accepted status line: %q", response)
	}

	if !strings.Contains(response, "Server: Flock/1.0\r\n") {
		t.Fatalf("response missing server header: %q", response)
	}

	if !strings.Contains(response, "X-Request-Id: "+strconv.FormatUint(ctx.id, 10)+"\r\n") {
		t.Fatalf("response missing request id header: %q", response)
	}

	if c.bytesWritten == 0 {
		t.Fatal("expected bytesWritten to increase")
	}
}

func TestConnectionSuccessWrite_HTTP10KeepAliveAddsConnectionHeader(t *testing.T) {
	conn := &recordingConn{}
	c := newTestConnection(conn)
	ctx := c.newRequestContext()
	ctx.version = protocol.HTTP10
	ctx.keepAlive = true

	c.successWrite(ctx, http.NewStatusResponse(protocol.StatusAccepted))

	response := conn.String()
	if !strings.Contains(response, "Connection: keep-alive\r\n") {
		t.Fatalf("response missing keep-alive header: %q", response)
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
			ctx := c.newRequestContext()

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
	ctx := c.newRequestContext()

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
	ctx := c.newRequestContext()

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
	ctx := c.newRequestContext()

	c.write(ctx, http.NewTextResponse(protocol.StatusOK, "hello"), nil)

	if c.bytesWritten == 0 {
		t.Fatal("expected bytesWritten to record partial write progress")
	}

	if conn.Len() == 0 {
		t.Fatal("expected partial response bytes to be written")
	}
}

func TestConnectionWrite_StripsBodyForNoContentResponses(t *testing.T) {
	conn := &recordingConn{}
	c := newTestConnection(conn)
	ctx := c.newRequestContext()

	c.write(ctx, http.NewTextResponse(protocol.StatusNoContent, "hello"), nil)

	response := conn.String()
	if strings.Contains(response, "hello") {
		t.Fatalf("expected no content response body to be stripped: %q", response)
	}

	if !strings.Contains(response, "Content-Length: 0\r\n") {
		t.Fatalf("expected content length zero: %q", response)
	}
}

func TestConnectionWrite_StripsBodyForNotModifiedResponses(t *testing.T) {
	conn := &recordingConn{}
	c := newTestConnection(conn)
	ctx := c.newRequestContext()

	c.write(ctx, http.NewTextResponse(protocol.StatusNotModified, "hello"), nil)

	response := conn.String()
	if strings.Contains(response, "hello") {
		t.Fatalf("expected not modified response body to be stripped: %q", response)
	}

	if !strings.Contains(response, "Content-Length: 0\r\n") {
		t.Fatalf("expected content length zero: %q", response)
	}
}

func TestConnectionWrite_AddsCloseHeaderWhenNotKeepingAlive(t *testing.T) {
	conn := &recordingConn{}
	c := newTestConnection(conn)
	ctx := c.newRequestContext()

	c.write(ctx, http.NewTextResponse(protocol.StatusOK, "hello"), nil)

	response := conn.String()
	if !strings.Contains(response, "Connection: close\r\n") {
		t.Fatalf("expected close header: %q", response)
	}
}
