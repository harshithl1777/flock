package http

import (
	"bufio"
	"encoding/json"
	"io"
	"strconv"

	"github.com/harshithl1777/flock/internal/errors"
	"github.com/harshithl1777/flock/internal/protocol"
)

type Response struct {
	StatusCode int
	StatusText string
	Headers    map[protocol.HeaderKey]string
	Body       string
}

// WithHeader stores or replaces a single response header.
func (r *Response) WithHeader(key protocol.HeaderKey, value string) *Response {
	r.Headers[key] = value
	return r
}

// WithBody replaces the response body.
//
// Content-Length is recomputed during WriteTo.
func (r *Response) WithBody(body string) *Response {
	r.Body = body
	return r
}

// WriteTo writes the response in HTTP/1.1 wire format to w.
//
// It preserves the current iteration order of response.Headers, omits any
// preexisting Content-Length entry, and appends a freshly computed
// Content-Length header immediately before the blank line and body.
func (response *Response) WriteTo(w io.Writer) (int64, error) {
	cw := &countingWriter{w: w}
	bw := bufio.NewWriter(cw)

	bw.WriteString(string(protocol.HTTP11))
	bw.WriteByte(' ')

	var b [20]byte // Uses a local buffer to avoid string allocation for status code
	bw.Write(strconv.AppendInt(b[:0], int64(response.StatusCode), 10))
	bw.WriteByte(' ')
	bw.WriteString(response.StatusText)
	bw.WriteString("\r\n")

	for headerKey, headerValue := range response.Headers {
		if headerKey == protocol.HeaderContentLength {
			continue
		}
		bw.WriteString(string(headerKey))
		bw.WriteString(": ")
		bw.WriteString(headerValue)
		bw.WriteString("\r\n")
	}

	bw.WriteString(string(protocol.HeaderContentLength) + ": ")
	bw.Write(strconv.AppendInt(b[:0], int64(len(response.Body)), 10))
	bw.WriteString("\r\n\r\n")

	if len(response.Body) > 0 {
		bw.WriteString(response.Body)
	}

	flushErr := bw.Flush()

	finalErr := cw.err
	if finalErr == nil {
		finalErr = flushErr
	}

	if finalErr != nil {
		return cw.count, errors.Wrap(errors.ConnectionWriteKind, "write response", finalErr)
	}

	return cw.count, nil
}

// NewTextResponse returns a text/plain response with the provided body.
func NewTextResponse(code protocol.StatusCode, body string) *Response {
	return newResponse(code).
		WithHeader(protocol.HeaderContentType, "text/plain; charset=utf-8").
		WithBody(body)
}

// NewHTMLResponse returns a text/html response with the provided body.
func NewHTMLResponse(code protocol.StatusCode, body string) *Response {
	return newResponse(code).
		WithHeader(protocol.HeaderContentType, "text/html; charset=utf-8").
		WithBody(body)
}

// NewJSONResponse returns a JSON response for the provided value.
func NewJSONResponse(code protocol.StatusCode, data interface{}) (*Response, *errors.OpError) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(errors.ResponseJSONSerializationKind, "serialize json", err)
	}

	return newResponse(code).
		WithHeader(protocol.HeaderContentType, "application/json; charset=utf-8").
		WithBody(string(body)), nil
}

// NewStatusResponse returns a response with no body-specific headers or payload.
func NewStatusResponse(code protocol.StatusCode) *Response {
	return newResponse(code).
		WithHeader(protocol.HeaderContentType, "text/plain; charset=utf-8").
		WithBody("")
}

// NewErrorResponse maps an OpError to a JSON HTTP error response.
func NewErrorResponse(err *errors.OpError) (*Response, *errors.OpError) {
	eb := struct {
		Kind        string `json:"error"`
		Description string `json:"description"`
	}{
		Kind:        err.Kind.String(),
		Description: err.Kind.Description(),
	}

	var code protocol.StatusCode

	switch err.Kind {
	case errors.MalformedRequestLineKind:
		code = protocol.StatusBadRequest
	case errors.MalformedHeaderKind:
		code = protocol.StatusBadRequest
	case errors.MissingHostKind:
		code = protocol.StatusBadRequest
	case errors.UnsupportedHTTPVersionKind:
		code = protocol.StatusHTTPVersionNotSupported
	case errors.UnsupportedHTTPMethodKind:
		code = protocol.StatusNotImplemented
	case errors.InvalidContentLengthKind:
		code = protocol.StatusBadRequest
	case errors.UnsupportedTransferEncodingKind:
		code = protocol.StatusBadRequest
	case errors.IncompleteBodyKind:
		code = protocol.StatusBadRequest
	case errors.RequestLineTooLargeKind:
		code = protocol.StatusBadRequest
	case errors.HeadersTooLargeKind:
		code = protocol.StatusRequestHeaderFieldsTooLarge
	case errors.MethodNotAllowedKind:
		code = protocol.StatusMethodNotAllowed
	case errors.BodyTooLargeKind:
		code = protocol.StatusBadRequest
	case errors.NotFoundKind:
		code = protocol.StatusNotFound
	default:
		code = protocol.StatusInternalServerError
	}

	return NewJSONResponse(code, eb)
}

// newResponse returns a response initialized with the supplied status code.
//
// It derives the HTTP reason phrase from code and allocates the headers map.
func newResponse(code protocol.StatusCode) *Response {
	const initialHeadersMapSize = 16
	return &Response{
		StatusCode: int(code),
		StatusText: code.Text(),
		Headers:    make(map[protocol.HeaderKey]string, initialHeadersMapSize),
		Body:       "",
	}
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
// Write forwards bytes to the underlying writer while tracking the first write error.
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
// WriteString forwards strings to the underlying writer while tracking the first write error.
func (cw *countingWriter) WriteString(s string) (int, error) {
	if cw.err != nil {
		return 0, cw.err
	}

	n, err := io.WriteString(cw.w, s)
	cw.count += int64(n)
	cw.err = err
	return n, cw.err
}
