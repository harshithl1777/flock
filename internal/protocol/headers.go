package protocol

type HeaderKey string

const (
	HeaderAllow            HeaderKey = "Allow"
	HeaderCacheControl     HeaderKey = "Cache-Control"
	HeaderLocation         HeaderKey = "Location"
	HeaderContentType      HeaderKey = "Content-Type"
	HeaderContentLength    HeaderKey = "Content-Length"
	HeaderConnection       HeaderKey = "Connection"
	HeaderHost             HeaderKey = "Host"
	HeaderServer           HeaderKey = "Server"
	HeaderTransferEncoding HeaderKey = "Transfer-Encoding"
	HeaderXRequestId       HeaderKey = "X-Request-Id"
)
