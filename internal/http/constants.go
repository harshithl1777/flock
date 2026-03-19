package http

import (
	"fmt"
)

type StatusCode int

// Status codes
const (
	StatusOK                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusNotFound            StatusCode = 404
	StatusInternalServerError StatusCode = 500
)

// statusText maps codes to their official HTTP strings
var statusText = map[StatusCode]string{
	StatusOK:                  "OK",
	StatusBadRequest:          "Bad Request",
	StatusNotFound:            "Not Found",
	StatusInternalServerError: "Internal Server Error",
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
	HeaderServer           HeaderKey = "Server"
	HeaderConnection       HeaderKey = "Connection"
	HeaderHost             HeaderKey = "Host"
	HeaderTransferEncoding HeaderKey = "Transfer-Encoding"
)

// StatusText returns the text for a StatusCode.
func (code StatusCode) Text() string {
	if text, ok := statusText[code]; ok {
		return fmt.Sprintf("%d %s", code, text)
	}
	return fmt.Sprintf("%d", code)
}

func (m Method) IsValid() bool {
	switch m {
	case "GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}

func (v Version) IsValid() bool {
	switch v {
	case "HTTP/1.0", "HTTP/1.1", "HTTP/2.0":
		return true
	default:
		return false
	}
}
