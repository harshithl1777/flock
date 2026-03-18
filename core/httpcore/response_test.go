package httpcore

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestWriteTo_DefaultResponse(t *testing.T) {
	response := NewResponse("Hello World!")
	var buf bytes.Buffer

	if _, err := response.WriteTo(&buf); err != nil {
		t.Fatalf("write response: %v", err)
	}

	got := buf.String()
	want := "" +
		"HTTP/1.1 200 OK\r\n" +
		"Connection: close\r\n" +
		"Content-Length: 12\r\n" +
		"Content-Type: text/plain\r\n" +
		"Server: Flock/1.0\r\n" +
		"\r\n" +
		"Hello World!"

	if got != want {
		t.Fatalf("serialized response mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestWriteTo_EmptyBody(t *testing.T) {
	response := NewResponse("")
	var buf bytes.Buffer

	if _, err := response.WriteTo(&buf); err != nil {
		t.Fatalf("write response: %v", err)
	}

	got := buf.String()
	want := "" +
		"HTTP/1.1 200 OK\r\n" +
		"Connection: close\r\n" +
		"Content-Length: 0\r\n" +
		"Content-Type: text/plain\r\n" +
		"Server: Flock/1.0\r\n" +
		"\r\n"

	if got != want {
		t.Fatalf("serialized response mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestWriteTo_CustomStatusAndHeaders(t *testing.T) {
	response := Response{
		StatusCode: 404,
		StatusText: "Not Found",
		Headers: map[string]string{
			"Content-Length": "9",
			"Content-Type":   "text/plain",
			"X-Trace-Id":     "abc123",
		},
		Body: "not found",
	}
	var buf bytes.Buffer

	if _, err := response.WriteTo(&buf); err != nil {
		t.Fatalf("write response: %v", err)
	}

	got := buf.String()
	want := "" +
		"HTTP/1.1 404 Not Found\r\n" +
		"Content-Length: 9\r\n" +
		"Content-Type: text/plain\r\n" +
		"X-Trace-Id: abc123\r\n" +
		"\r\n" +
		"not found"

	if got != want {
		t.Fatalf("serialized response mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestWriteTo_RecomputesContentLength(t *testing.T) {
	response := NewResponse("Hello")
	response.Body = "Hello World!"
	var buf bytes.Buffer

	n, err := response.WriteTo(&buf)
	if err != nil {
		t.Fatalf("write response: %v", err)
	}

	got := buf.String()

	if n != int64(buf.Len()) {
		t.Fatalf("write count mismatch: got %d, want %d", n, buf.Len())
	}

	if !strings.Contains(got, "Content-Length: 12\r\n") {
		t.Fatalf("expected recomputed Content-Length header, got:\n%q", got)
	}
}

func TestWriteTo_OverridesStaleContentLengthHeader(t *testing.T) {
	response := NewResponse("Hello World!")
	response.Headers["Content-Length"] = "999"
	var buf bytes.Buffer

	if _, err := response.WriteTo(&buf); err != nil {
		t.Fatalf("write response: %v", err)
	}

	got := buf.String()

	if !strings.Contains(got, "Content-Length: 12\r\n") {
		t.Fatalf("expected recomputed Content-Length header, got:\n%q", got)
	}

	if strings.Contains(got, "Content-Length: 999\r\n") {
		t.Fatalf("expected stale Content-Length header to be removed, got:\n%q", got)
	}
}

type failAfterNWriter struct {
	remaining int
	err       error
}

func (w *failAfterNWriter) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		return 0, w.err
	}

	if len(p) > w.remaining {
		n := w.remaining
		w.remaining = 0
		return n, w.err
	}

	w.remaining -= len(p)
	return len(p), nil
}

var _ io.Writer = (*failAfterNWriter)(nil)

func TestWriteTo_PropagatesWriterErrorAndPartialCount(t *testing.T) {
	response := NewResponse("Hello World!")
	response.Headers["Content-Length"] = "999"

	expected := errors.New("write failed")
	writer := &failAfterNWriter{
		remaining: 16,
		err:       expected,
	}

	// WriteTo writes through countingWriter, so its returned (n, err) should
	// reflect the underlying writer's partial progress and original failure.
	n, err := response.WriteTo(writer)

	if !errors.Is(err, expected) {
		t.Fatalf("expected WriteTo to return the original writer error, got %v", err)
	}

	if n != 16 {
		t.Fatalf("expected WriteTo to report the partial byte count from countingWriter, got %d", n)
	}
}
