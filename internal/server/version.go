package server

import "fmt"

const (
	ServerName    = "Flock"
	ServerVersion = "1.0"
)

// FullServerVersion returns the string used in the "Server" HTTP header.
func FullServerVersion() string {
	return fmt.Sprintf("%s/%s", ServerName, ServerVersion)
}
