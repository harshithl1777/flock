package httpcore

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
)

type Response struct {
	StatusCode int
	StatusText string
	Headers    map[string]string
	Body       string
}

// NewResponse returns a default plain-text HTTP 200 response.
//
// It initializes the standard headers used by the server and stores the
// supplied body.
func NewResponse(body string) Response {
	return Response{
		StatusCode: 200,
		StatusText: "OK",
		Headers: map[string]string{
			"Content-Type": "text/plain",
			"Connection":   "close",
			"Server":       "Flock/1.0",
		},
		Body: body,
	}
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
		if key == contentLengthHeaderKey {
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

// SerializeResponse serializes the response into HTTP/1.1 wire format.
//
// It recomputes the Content-Length header from the current body so stale
// header values do not appear in the serialized output.
func (response *Response) SerializeResponse() []byte {
	headers, headerKeys := computePrewriteHeadersAndSortedKeys(response)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "HTTP/1.1 %d %s\r\n", response.StatusCode, response.StatusText)

	for _, key := range headerKeys {
		value := headers[key]
		fmt.Fprintf(&buf, "%s: %s\r\n", key, value)
	}

	buf.WriteString("\r\n")
	buf.WriteString(response.Body)

	return buf.Bytes()
}
