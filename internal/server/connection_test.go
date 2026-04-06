package server

import (
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/protocol"
	"github.com/harshithl1777/flock/internal/router"
)

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

func TestConnectionServe_WritesHTTPResponse(t *testing.T) {
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
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	go NewConnection(serverConn, testRouter()).serve()

	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}

	request := "" +
		"GET / HTTP/1.1\r\n" +
		"\r\n"
	if _, err := clientConn.Write([]byte(request)); err != nil {
		t.Fatalf("write request: %v", err)
	}

	responseBytes, err := io.ReadAll(clientConn)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	response := string(responseBytes)

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
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	go NewConnection(serverConn, testRouter()).serve()

	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}

	request := "" +
		"POST / HTTP/1.1\r\n" +
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

	if !strings.HasPrefix(response, "HTTP/1.1 405 Method Not Allowed\r\n") {
		t.Fatalf("response missing method not allowed status line: %q", response)
	}

	if !strings.Contains(response, "Allow: GET, HEAD\r\n") {
		t.Fatalf("response missing allow header: %q", response)
	}
}

func TestConnectionServe_OptionsReturnsNoContentAndAllowHeader(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	go NewConnection(serverConn, testRouter()).serve()

	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}

	request := "" +
		"OPTIONS / HTTP/1.1\r\n" +
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

	if !strings.HasPrefix(response, "HTTP/1.1 204 No Content\r\n") {
		t.Fatalf("response missing no content status line: %q", response)
	}

	if !strings.Contains(response, "Allow: GET, HEAD\r\n") {
		t.Fatalf("response missing allow header: %q", response)
	}
}
