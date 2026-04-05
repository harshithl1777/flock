package errors

type ErrorKind int

const (
	ConfigLoadKind ErrorKind = iota
	ServerStartupKind
	MalformedRequestLineKind
	MalformedHeaderKind
	MissingHostKind
	UnsupportedHTTPMethodKind
	UnsupportedHTTPVersionKind
	InvalidContentLengthKind
	UnsupportedTransferEncodingKind
	IncompleteBodyKind
	RequestLineTooLargeKind
	HeadersTooLargeKind
	BodyTooLargeKind
	MethodNotAllowedKind
	NotFoundKind
	ConnectionWriteKind
	ResponseJSONSerializationKind
	InternalServerErrorKind
)

// String returns the stable identifier used for logs and API responses.
func (k ErrorKind) String() string {
	switch k {
	case ServerStartupKind:
		return "server_startup"
	case ConfigLoadKind:
		return "config_load"
	case MalformedRequestLineKind:
		return "malformed_request_line"
	case MalformedHeaderKind:
		return "malformed_header"
	case MissingHostKind:
		return "missing_host"
	case InvalidContentLengthKind:
		return "invalid_content_length"
	case UnsupportedHTTPMethodKind:
		return "unsupported_http_method"
	case UnsupportedHTTPVersionKind:
		return "unsupported_http_version"
	case UnsupportedTransferEncodingKind:
		return "unsupported_transfer_encoding"
	case IncompleteBodyKind:
		return "incomplete_body"
	case RequestLineTooLargeKind:
		return "request_line_too_large"
	case HeadersTooLargeKind:
		return "headers_too_large"
	case BodyTooLargeKind:
		return "body_too_large"
	case MethodNotAllowedKind:
		return "method_not_allowed"
	case NotFoundKind:
		return "not_found"
	case ConnectionWriteKind:
		return "connection_write"
	case InternalServerErrorKind:
		return "internal_server_error"
	case ResponseJSONSerializationKind:
		return "response_json_serialization"
	default:
		return "unknown"
	}
}

// Description returns the human-readable explanation for an error kind.
func (k ErrorKind) Description() string {
	switch k {
	case ConfigLoadKind:
		return "failed to load configuration"
	case ServerStartupKind:
		return "failed to start the server"
	case MalformedRequestLineKind:
		return "the request line is malformed"
	case MalformedHeaderKind:
		return "one or more headers are malformed"
	case MissingHostKind:
		return "the Host header is required"
	case InvalidContentLengthKind:
		return "invalid Content-Length header"
	case UnsupportedHTTPMethodKind:
		return "HTTP method not supported"
	case UnsupportedHTTPVersionKind:
		return "HTTP version not supported"
	case UnsupportedTransferEncodingKind:
		return "Transfer-Encoding not supported"
	case IncompleteBodyKind:
		return "request body is incomplete"
	case RequestLineTooLargeKind:
		return "request line exceeds size limit"
	case HeadersTooLargeKind:
		return "request headers exceed size limit"
	case BodyTooLargeKind:
		return "request body exceeds size limit"
	case MethodNotAllowedKind:
		return "requested method not allowed on this route"
	case NotFoundKind:
		return "requested resource not found"
	case ConnectionWriteKind:
		return "failed to write to connection"
	case ResponseJSONSerializationKind:
		return "failed to serialize response as JSON"
	default:
		return "an unknown error occurred"
	}
}
