package server

import (
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestHandleConnection_WritesHTTPResponse(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	srv := &Server{}

	go srv.handleConnection(serverConn)

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

	if !strings.Contains(response, "\r\n\r\nHello World!") {
		t.Fatalf("response missing body separator or body: %q", response)
	}

	if !strings.Contains(response, "Content-Length: 12\r\n") {
		t.Fatalf("response missing content length: %q", response)
	}
}
