package handler

import (
	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/http"
	"github.com/harshithl1777/flock/internal/protocol"
)

// StatusHandler responds with the configured status code and no body.
type StatusHandler struct {
	code protocol.StatusCode
}

// Handle returns an empty response with the configured status code.
func (h *StatusHandler) Handle(req *http.Request) *http.Response {
	return http.NewStatusResponse(h.code)
}

// Name returns the stable handler identifier used in logs.
func (h *StatusHandler) Name() string {
	return "status"
}

// NewStatusHandler constructs a status handler from validated config options.
func NewStatusHandler(options *config.HandlerStatusOptions) Handler {
	return &StatusHandler{code: options.Code}
}
