package httpcore

import "testing"

func TestSerializeResponse_DefaultResponse(t *testing.T) {
	response := NewResponse("Hello World!")

	got := string(response.SerializeResponse())
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

func TestSerializeResponse_EmptyBody(t *testing.T) {
	response := NewResponse("")

	got := string(response.SerializeResponse())
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

func TestSerializeResponse_CustomStatusAndHeaders(t *testing.T) {
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

	got := string(response.SerializeResponse())
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
