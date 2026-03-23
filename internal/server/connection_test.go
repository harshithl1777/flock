package server

import (
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestConnectionServe_WritesHTTPResponse(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	go newConnection(serverConn).serve()

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

	responseBytes, err := io.ReadAll(clientConn)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	response := string(responseBytes)

	if !strings.HasPrefix(response, "HTTP/1.1 200 OK\r\n") {
		t.Fatalf("response missing status line: %q", response)
	}

	if !strings.Contains(response, "Content-Type: text/plain; charset=utf-8\r\n") {
		t.Fatalf("response missing content type: %q", response)
	}

	if !strings.Contains(response, "Connection: close\r\n") {
		t.Fatalf("response missing connection header: %q", response)
	}

	if !strings.Contains(response, "Server: Flock/1.0\r\n") {
		t.Fatalf("response missing server header: %q", response)
	}

	if !strings.Contains(response, "X-Request-Id: req_") {
		t.Fatalf("response missing request id: %q", response)
	}

	if !strings.Contains(response, "Content-Length: 0\r\n") {
		t.Fatalf("response missing content length: %q", response)
	}
}
