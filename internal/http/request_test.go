package http

import (
	"bufio"
	stderrors "errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/harshithl1777/flock/internal/errors"
	"github.com/harshithl1777/flock/internal/protocol"
)

func assertRequestErrorKind(t *testing.T, err error, want errors.ErrorKind) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var opErr *errors.OpError
	if !stderrors.As(err, &opErr) {
		t.Fatalf("expected *errors.OpError, got %T", err)
	}

	if opErr.Kind != want {
		t.Fatalf("got error kind %s, want %s", opErr.Kind, want)
	}
}

func TestReadRequest_ParsesHeadersAndBody(t *testing.T) {
	raw := "" +
		"POST /submit HTTP/1.1\r\n" +
		"host: localhost\r\n" +
		"content-length: 5\r\n" +
		"x-trace-id: abc123\r\n" +
		"\r\n" +
		"hello"

	request, err := ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatalf("ReadRequest returned error: %v", err)
	}

	if request.Method != protocol.Post {
		t.Fatalf("got method %q, want %q", request.Method, protocol.Post)
	}

	if request.Path != "/submit" {
		t.Fatalf("got path %q, want /submit", request.Path)
	}

	if request.Version != protocol.HTTP11 {
		t.Fatalf("got version %q, want %q", request.Version, protocol.HTTP11)
	}

	if got := request.Headers["Host"]; got != "localhost" {
		t.Fatalf("got Host header %q, want localhost", got)
	}

	if got := request.Headers["Content-Length"]; got != "5" {
		t.Fatalf("got Content-Length header %q, want 5", got)
	}

	if got := request.Headers["X-Trace-Id"]; got != "abc123" {
		t.Fatalf("got X-Trace-Id header %q, want abc123", got)
	}

	if got := string(request.Body); got != "hello" {
		t.Fatalf("got body %q, want hello", got)
	}
}

func TestReadRequest_InvalidRequestLine(t *testing.T) {
	raw := "TRACE / HTTP/1.1\r\nHost: localhost\r\n\r\n"

	_, err := ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	assertRequestErrorKind(t, err, errors.UnsupportedHTTPMethodKind)
}

func TestReadRequest_RejectsUnsupportedHTTPVersion(t *testing.T) {
	raw := "GET / HTTP/2.0\r\nHost: localhost\r\n\r\n"

	_, err := ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	assertRequestErrorKind(t, err, errors.UnsupportedHTTPVersionKind)
}

func TestReadRequest_RejectsChunkedTransferEncoding(t *testing.T) {
	raw := "" +
		"POST /submit HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\n"

	_, err := ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	assertRequestErrorKind(t, err, errors.UnsupportedTransferEncodingKind)
}

func TestReadRequest_RejectsMalformedHeader(t *testing.T) {
	raw := "" +
		"GET / HTTP/1.1\r\n" +
		"Host localhost\r\n" +
		"\r\n"

	_, err := ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	assertRequestErrorKind(t, err, errors.MalformedHeaderKind)
}

func TestReadRequest_RejectsEmptyHeaderKey(t *testing.T) {
	raw := "" +
		"GET / HTTP/1.1\r\n" +
		": localhost\r\n" +
		"\r\n"

	_, err := ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	assertRequestErrorKind(t, err, errors.MalformedHeaderKind)
}

func TestReadRequest_RequiresHostHeader(t *testing.T) {
	raw := "" +
		"GET / HTTP/1.1\r\n" +
		"\r\n"

	_, err := ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	assertRequestErrorKind(t, err, errors.MissingHostKind)
}

func TestReadRequest_HTTP10DoesNotRequireHostHeader(t *testing.T) {
	raw := "" +
		"GET / HTTP/1.0\r\n" +
		"\r\n"

	request, err := ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatalf("ReadRequest returned error: %v", err)
	}

	if request.Version != protocol.HTTP10 {
		t.Fatalf("got version %q, want %q", request.Version, protocol.HTTP10)
	}
}

func TestReadRequest_RejectsInvalidContentLength(t *testing.T) {
	raw := "" +
		"POST /submit HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"Content-Length: nope\r\n" +
		"\r\n"

	_, err := ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	assertRequestErrorKind(t, err, errors.InvalidContentLengthKind)
}

func TestReadRequest_RejectsIncompleteBody(t *testing.T) {
	raw := "" +
		"POST /submit HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"Content-Length: 5\r\n" +
		"\r\n" +
		"hey"

	_, err := ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	assertRequestErrorKind(t, err, errors.IncompleteBodyKind)
}

func TestReadRequest_ClientClosedBeforeRequestLine(t *testing.T) {
	_, err := ReadRequest(bufio.NewReader(strings.NewReader("")))
	assertRequestErrorKind(t, err, errors.ClientClosedConnectionKind)
}

func TestReadRequest_TimeoutBeforeRequestLine(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	if err := serverConn.SetReadDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	_, err := ReadRequest(bufio.NewReader(serverConn))
	assertRequestErrorKind(t, err, errors.RequestTimeoutKind)
}

func TestReadRequest_TimeoutWhileReadingHeaders(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	go func() {
		_, _ = clientConn.Write([]byte("GET / HTTP/1.1\r\n"))
	}()

	if err := serverConn.SetReadDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	_, err := ReadRequest(bufio.NewReader(serverConn))
	assertRequestErrorKind(t, err, errors.RequestTimeoutKind)
}

func TestReadRequest_TimeoutWhileReadingBody(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	go func() {
		raw := "" +
			"POST /submit HTTP/1.1\r\n" +
			"Host: localhost\r\n" +
			"Content-Length: 5\r\n" +
			"\r\n" +
			"he"
		_, _ = clientConn.Write([]byte(raw))
	}()

	if err := serverConn.SetReadDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	_, err := ReadRequest(bufio.NewReader(serverConn))
	assertRequestErrorKind(t, err, errors.RequestTimeoutKind)
}
