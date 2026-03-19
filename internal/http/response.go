package http

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

type Response struct {
	StatusCode int
	StatusText string
	Headers    map[string]string
	Body       string
}

// NewResponse returns a plain-text response initialized for code and body.
//
// It derives the HTTP reason phrase from code, initializes the standard
// headers used by the server, and stores the supplied body.
func NewResponse(code StatusCode, body string) *Response {
	return &Response{
		StatusCode: int(code),
		StatusText: statusText[code],
		Headers: map[string]string{
			"Content-Type": "text/plain",
			"Connection":   "close",
			"Server":       "Flock/1.0",
		},
		Body: body,
	}
}

// WriteTo writes the response in HTTP/1.1 wire format to w.
//
// It preserves the current iteration order of response.Headers, omits any
// preexisting Content-Length entry, and appends a freshly computed
// Content-Length header immediately before the blank line and body.
func (response *Response) WriteTo(w io.Writer) (int64, error) {
	cw := &countingWriter{w: w}
	bw := bufio.NewWriter(cw)

	bw.WriteString(string(HTTP11))
	bw.WriteByte(' ')

	var b [10]byte // Uses a local buffer to avoid string allocation for status code
	bw.Write(strconv.AppendInt(b[:0], int64(response.StatusCode), 10))
	bw.WriteByte(' ')
	bw.WriteString(response.StatusText)
	bw.WriteString("\r\n")

	for k, v := range response.Headers {
		if strings.EqualFold(k, "Content-Length") {
			continue
		}
		bw.WriteString(k)
		bw.WriteString(": ")
		bw.WriteString(v)
		bw.WriteString("\r\n")
	}

	bw.WriteString("Content-Length: ")
	bw.Write(strconv.AppendInt(b[:0], int64(len(response.Body)), 10))
	bw.WriteString("\r\n\r\n")

	bw.WriteString(response.Body)

	err := bw.Flush()
	if cw.err == nil {
		cw.err = err
	}
	return cw.count, cw.err
}

var _ io.Writer = (*countingWriter)(nil)
var _ io.StringWriter = (*countingWriter)(nil)

// countingWriter forwards writes while tracking the total bytes written and the
// first error returned by the underlying writer.
type countingWriter struct {
	w     io.Writer
	count int64
	err   error
}

// Write implements the io.Writer interface.
func (cw *countingWriter) Write(p []byte) (int, error) {
	if cw.err != nil {
		return 0, cw.err
	}

	n, err := cw.w.Write(p)
	cw.count += int64(n)
	cw.err = err
	return n, cw.err
}

// WriteString implements the io.StringWriter interface.
func (cw *countingWriter) WriteString(s string) (int, error) {
	if cw.err != nil {
		return 0, cw.err
	}

	n, err := io.WriteString(cw.w, s)
	cw.count += int64(n)
	cw.err = err
	return n, cw.err
}
