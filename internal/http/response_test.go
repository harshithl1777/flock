package http

import (
	"bytes"
	stdErrors "errors"
	"io"
	"strings"
	"testing"

	flockerrors "github.com/harshithl1777/flock/internal/errors"
)

func assertResponseHasParts(t *testing.T, got string, parts ...string) {
	t.Helper()

	for _, part := range parts {
		if !strings.Contains(got, part) {
			t.Fatalf("response missing %q:\n%q", part, got)
		}
	}
}

func TestWriteTo_TextResponse(t *testing.T) {
	response := NewTextResponse(StatusOK, "Hello World!")
	var buf bytes.Buffer

	if _, err := response.WriteTo(&buf); err != nil {
		t.Fatalf("write response: %v", err)
	}

	got := buf.String()
	assertResponseHasParts(t, got,
		"HTTP/1.1 200 OK\r\n",
		"Content-Type: text/plain; charset=utf-8\r\n",
		"Content-Length: 12\r\n",
		"\r\n\r\nHello World!",
	)
}

func TestWriteTo_EmptyBody(t *testing.T) {
	response := NewStatusResponse(StatusOK)
	var buf bytes.Buffer

	if _, err := response.WriteTo(&buf); err != nil {
		t.Fatalf("write response: %v", err)
	}

	got := buf.String()
	assertResponseHasParts(t, got,
		"HTTP/1.1 200 OK\r\n",
		"Content-Length: 0\r\n",
		"\r\n\r\n",
	)

	if !strings.HasSuffix(got, "\r\n\r\n") {
		t.Fatalf("expected empty-body response terminator, got %q", got)
	}
}

func TestWriteTo_RecomputesContentLength(t *testing.T) {
	response := NewTextResponse(StatusOK, "Hello")
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
	response := NewTextResponse(StatusOK, "Hello World!")
	response.Headers[HeaderContentLength] = "999"
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

func TestNewJSONResponse_SerializesBody(t *testing.T) {
	response, err := NewJSONResponse(StatusOK, map[string]string{"hello": "world"})
	if err != nil {
		t.Fatalf("NewJSONResponse returned error: %v", err)
	}

	if got := response.Headers[HeaderContentType]; got != "application/json; charset=utf-8" {
		t.Fatalf("got content type %q, want application/json; charset=utf-8", got)
	}

	if got := response.Body; got != `{"hello":"world"}` {
		t.Fatalf("got body %q, want serialized json", got)
	}
}

func TestNewErrorResponse_MapsBodyTooLargeToBadRequest(t *testing.T) {
	response, err := NewErrorResponse(flockerrors.New(flockerrors.BodyTooLargeKind, "parse body", "too large"))
	if err != nil {
		t.Fatalf("NewErrorResponse returned error: %v", err)
	}

	if response.StatusCode != int(StatusBadRequest) {
		t.Fatalf("got status %d, want %d", response.StatusCode, StatusBadRequest)
	}

	if !strings.Contains(response.Body, `"error":"body_too_large"`) {
		t.Fatalf("expected error kind in body, got %q", response.Body)
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
	response := NewTextResponse(StatusOK, "Hello World!")
	response.Headers[HeaderContentLength] = "999"

	expected := stdErrors.New("write failed")
	writer := &failAfterNWriter{
		remaining: 16,
		err:       expected,
	}

	n, err := response.WriteTo(writer)

	if !stdErrors.Is(err, expected) {
		t.Fatalf("expected WriteTo to return the original writer error, got %v", err)
	}

	if n != 16 {
		t.Fatalf("expected WriteTo to report the partial byte count from countingWriter, got %d", n)
	}
}
