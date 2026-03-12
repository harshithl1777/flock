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

func (response *Response) SerializeResponse() []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "HTTP/1.1 %d %s\r\n", response.StatusCode, response.StatusText)

	keys := make([]string, 0, len(response.Headers))
	for key := range response.Headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := response.Headers[key]
		fmt.Fprintf(&buf, "%s: %s\r\n", key, value)
	}

	buf.WriteString("\r\n")
	buf.WriteString(response.Body)

	return buf.Bytes()
}
