package httpcore

import (
	"bufio"
	"io"
	"sort"
	"strconv"
	"strings"
)

type Response struct {
	StatusCode int
	StatusText string
	Headers    map[string]string
	Body       string
}

// NewResponse returns a default plain-text HTTP 200 response.
//
// It initializes the standard headers used by the server and stores the supplied body.
// Content-Length is populated from the body and recomputed when the response is written.
func NewResponse(body string) Response {
	return Response{
		StatusCode: 200,
		StatusText: "OK",
		Headers: map[string]string{
			"Content-Type":   "text/plain",
			"Connection":     "close",
			"Server":         "Flock/1.0",
			"Content-Length": strconv.Itoa(len(body)),
		},
		Body: body,
	}
}

// WriteTo writes the response in HTTP/1.1 wire format to w.
//
// It recomputes the Content-Length header from the current body so stale
// header values do not appear in the serialized output.
func (response *Response) WriteTo(w io.Writer) (int64, error) {
	cw := &countingWriter{w: w}
	bw := bufio.NewWriter(cw) // Buffers the output for smaller strings to reduce syscalls

	headers, headerKeys := computePrewriteHeadersAndSortedKeys(response)
	bw.WriteString("HTTP/1.1 ")
	bw.WriteString(strconv.Itoa(response.StatusCode))
	bw.WriteString(" ")
	bw.WriteString(response.StatusText)
	bw.WriteString("\r\n")

	for _, key := range headerKeys {
		bw.WriteString(key)
		bw.WriteString(": ")
		bw.WriteString(headers[key])
		bw.WriteString("\r\n")
	}

	bw.WriteString("\r\n")
	bw.WriteString(response.Body)

	if err := bw.Flush(); err != nil && cw.err == nil {
		cw.err = err
	}

	return cw.count, cw.err
}

// computePrewriteHeadersAndSortedKeys prepares headers for serialization.
//
// It returns a copy of the response headers with a freshly computed
// Content-Length and the corresponding sorted header keys.
func computePrewriteHeadersAndSortedKeys(response *Response) (map[string]string, []string) {
	headers := make(map[string]string, len(response.Headers)+1)
	headerKeys := make([]string, 0, len(response.Headers)+1)

	const contentLengthHeaderKey = "Content-Length"
	for key, value := range response.Headers {
		if strings.EqualFold(key, contentLengthHeaderKey) {
			continue
		}

		headers[key] = value
		headerKeys = append(headerKeys, key)
	}

	headers[contentLengthHeaderKey] = strconv.Itoa(len(response.Body))
	headerKeys = append(headerKeys, contentLengthHeaderKey)
	sort.Strings(headerKeys)

	return headers, headerKeys
}

var _ io.Writer = (*countingWriter)(nil)
var _ io.StringWriter = (*countingWriter)(nil)

// countingWriter intercepts writes to an underlying io.Writer to track the total number of
// bytes written and capture the first error encountered.
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
