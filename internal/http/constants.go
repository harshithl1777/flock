package http

import (
	"fmt"
)

type StatusCode int

// Status codes
const (
	StatusOK                          StatusCode = 200
	StatusBadRequest                  StatusCode = 400
	StatusNotFound                    StatusCode = 404
	StatusRequestHeaderFieldsTooLarge StatusCode = 431
	StatusInternalServerError         StatusCode = 500
	StatusHTTPVersionNotSupported     StatusCode = 505
)

// statusText maps codes to their official HTTP strings
var statusText = map[StatusCode]string{
	StatusOK:                          "OK",
	StatusBadRequest:                  "Bad Request",
	StatusNotFound:                    "Not Found",
	StatusRequestHeaderFieldsTooLarge: "Request Header Fields Too Large",
	StatusInternalServerError:         "Internal Server Error",
	StatusHTTPVersionNotSupported:     "HTTP Version Not Supported",
}

type Method string

const (
	Head    Method = "HEAD"
	Get     Method = "GET"
	Post    Method = "POST"
	Put     Method = "PUT"
	Patch   Method = "PATCH"
	Delete  Method = "DELETE"
	Options Method = "OPTIONS"
)

type Version string

const (
	HTTP10 Version = "HTTP/1.0"
	HTTP11 Version = "HTTP/1.1"
	HTTP20 Version = "HTTP/2.0"
)

type HeaderKey string

const (
	HeaderContentType      HeaderKey = "Content-Type"
	HeaderContentLength    HeaderKey = "Content-Length"
	HeaderConnection       HeaderKey = "Connection"
	HeaderHost             HeaderKey = "Host"
	HeaderServer           HeaderKey = "Server"
	HeaderTransferEncoding HeaderKey = "Transfer-Encoding"
	HeaderXRequestId       HeaderKey = "X-Request-Id"
)

// StatusText returns the text for a StatusCode.
func (code StatusCode) Text() string {
	if text, ok := statusText[code]; ok {
		return fmt.Sprintf("%d %s", code, text)
	}
	return fmt.Sprintf("%d", code)
}

// IsValid reports whether m is one of the supported HTTP methods.
func (m Method) IsValid() bool {
	switch m {
	case "GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}

// IsValid reports whether v is one of the supported HTTP versions.
func (v Version) IsValid() bool {
	switch v {
	case "HTTP/1.0", "HTTP/1.1":
		return true
	default:
		return false
	}
}
