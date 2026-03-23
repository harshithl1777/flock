package http

import (
	"bufio"
	"io"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/harshithl1777/flock/internal/errors"
)

type Request struct {
	Method  Method
	Path    string
	Version Version
	Headers map[string]string
	Body    []byte
}

// ReadRequest parses a single HTTP request from reader.
//
// It reads the request line, headers, and optional fixed-length body, then
// returns the normalized request values.
func ReadRequest(reader *bufio.Reader) (*Request, *errors.OpError) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, errors.Wrap(errors.MalformedRequestLineKind, "read request", err)
	}

	method, path, version, opErr := parseRequestLine(line)
	if opErr != nil {
		return nil, errors.Chain("read request", opErr)
	}

	headers, opErr := parseHeaders(reader)
	if opErr != nil {
		return nil, errors.Chain("read request", opErr)
	}

	body, opErr := readBody(reader, headers)
	if opErr != nil {
		return nil, errors.Chain("read request", opErr)
	}

	return &Request{
		Method:  method,
		Path:    path,
		Version: version,
		Headers: headers,
		Body:    body,
	}, nil
}

// parseRequestLine validates and splits a single HTTP request line.
//
// The line must contain exactly a method, path, and version.
func parseRequestLine(line string) (Method, string, Version, *errors.OpError) {
	line = strings.TrimRight(line, "\r\n")
	parts := strings.Split(line, " ")

	if len(parts) != 3 {
		return "", "", "", errors.Newf(errors.MalformedRequestLineKind, "parse request line", "malformed request line: %s", line)
	}

	method := Method(parts[0])
	path := parts[1]
	version := Version(parts[2])

	if !method.IsValid() {
		return "", "", "", errors.Newf(errors.UnsupportedHTTPMethodKind, "parse request line", "invalid http method: %s", method)
	} else if !version.IsValid() {
		return "", "", "", errors.Newf(errors.UnsupportedHTTPVersionKind, "parse request line", "invalid http version: %s", version)
	}

	return method, path, version, nil
}

// parseHeaders reads header lines until the terminating blank line.
//
// Header keys are canonicalized using MIME header casing, and malformed lines
// without a separating colon are ignored.
func parseHeaders(reader *bufio.Reader) (map[string]string, *errors.OpError) {
	const maxHeaders = 100
	const initialHeadersMapSize = 16

	headers := make(map[string]string, initialHeadersMapSize)
	count := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, errors.Wrap(errors.MalformedHeaderKind, "parse headers", err)
		}

		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			break
		}

		count++
		if count > maxHeaders {
			return nil, errors.New(errors.HeadersTooLargeKind, "parse headers", "too many headers received")
		}

		colonIndex := strings.IndexByte(line, ':')
		if colonIndex <= 0 { // TODO: do not skip, reject with 400
			return nil, errors.New(errors.MalformedHeaderKind, "parse headers", "missing header key-value pair colon")
		}

		key := strings.TrimSpace(line[:colonIndex])
		value := strings.TrimSpace(line[colonIndex+1:])

		headers[textproto.CanonicalMIMEHeaderKey(key)] = value
	}

	return headers, nil
}

// readBody reads the request body when Content-Length is present.
//
// It currently supports only fixed-length bodies and rejects chunked transfer
// encoding and bodies larger than the in-memory safety limit.
func readBody(reader *bufio.Reader, headers map[string]string) ([]byte, *errors.OpError) {
	if headers[string(HeaderTransferEncoding)] == "chunked" {
		return nil, errors.New(errors.UnsupportedTransferEncodingKind, "parse body", "chunked encoding not supported")
	}

	contentLengthStr := headers[string(HeaderContentLength)]
	if contentLengthStr == "" {
		return nil, nil
	}

	contentLength, err := strconv.Atoi(contentLengthStr)
	if err != nil || contentLength < 0 {
		return nil, errors.Newf(errors.InvalidContentLengthKind, "parse body", "invalid content-length: %s", contentLengthStr)
	}

	const MaxBodyReadSize = 2 * 1024 * 1024
	if contentLength > MaxBodyReadSize {
		return nil, errors.Newf(errors.BodyTooLargeKind, "parse body", "content-length exceeds limit: %d", contentLength)
	}

	body := make([]byte, contentLength)
	_, err = io.ReadFull(reader, body)
	if err != nil {
		return nil, errors.Wrap(errors.IncompleteBodyKind, "parse body", err)
	}

	return body, nil
}
