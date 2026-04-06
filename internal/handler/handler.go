package handler

import (
	"github.com/harshithl1777/flock/internal/http"
)

// Handler is the runtime contract implemented by all route handlers.
type Handler interface {
	Handle(req *http.Request) *http.Response
	Name() string
}
