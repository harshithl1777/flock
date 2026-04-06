package protocol

import "fmt"

const (
	ServerName    = "Flock"
	ServerVersion = "1.0"
)

// FullServerVersion returns the string used in the "Server" HTTP header and
// health responses.
func FullServerVersion() string {
	return fmt.Sprintf("%s/%s", ServerName, ServerVersion)
}
