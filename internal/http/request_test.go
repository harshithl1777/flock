package http

import (
	"bufio"
	"strings"
	"testing"
)

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

	if request.Method != Post {
		t.Fatalf("got method %q, want %q", request.Method, Post)
	}

	if request.Path != "/submit" {
		t.Fatalf("got path %q, want /submit", request.Path)
	}

	if request.Version != HTTP11 {
		t.Fatalf("got version %q, want %q", request.Version, HTTP11)
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
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if got := err.Error(); !strings.Contains(got, "invalid http method: TRACE") {
		t.Fatalf("unexpected error: %v", got)
	}
}

func TestReadRequest_RejectsChunkedTransferEncoding(t *testing.T) {
	raw := "" +
		"POST /submit HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\n"

	_, err := ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if got := err.Error(); !strings.Contains(got, "chunked encoding not supported") {
		t.Fatalf("unexpected error: %v", got)
	}
}
