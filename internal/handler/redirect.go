package handler

import (
	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/http"
	"github.com/harshithl1777/flock/internal/protocol"
)

// RedirectHandler responds with a redirect status and Location header.
type RedirectHandler struct {
	code        protocol.StatusCode
	destination string
}

// Handle returns an empty redirect response with the configured destination.
func (h *RedirectHandler) Handle(req *http.Request) *http.Response {
	return http.NewStatusResponse(h.code).
		WithHeader(protocol.HeaderLocation, h.destination)
}

// Name returns the stable handler identifier used in logs.
func (h *RedirectHandler) Name() string {
	return "redirect"
}

// NewRedirectHandler constructs a redirect handler from validated config options.
func NewRedirectHandler(options *config.HandlerRedirectOptions) Handler {
	return &RedirectHandler{code: options.Code, destination: options.Destination}
}
