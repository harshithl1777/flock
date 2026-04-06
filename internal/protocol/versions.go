package protocol

type Version string

const (
	HTTP10 Version = "HTTP/1.0"
	HTTP11 Version = "HTTP/1.1"
	HTTP20 Version = "HTTP/2.0"
)

// IsValid reports whether v is one of the supported HTTP versions.
func (v Version) IsValid() bool {
	switch v {
	case "HTTP/1.0", "HTTP/1.1":
		return true
	default:
		return false
	}
}
