package handler

import (
	"time"

	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/http"
	"github.com/harshithl1777/flock/internal/protocol"
)

// HealthHandler serves the configured health endpoint.
type HealthHandler struct {
	code protocol.StatusCode
}

// HealthJSON is the structured payload returned by the health handler.
type HealthJSON struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}

var healthStartedAt = time.Now()

// Handle returns the health payload with a no-cache policy.
func (h *HealthHandler) Handle(req *http.Request) *http.Response {
	response, err := http.NewJSONResponse(h.code, HealthJSON{
		Status:  "pass",
		Version: protocol.ServerVersion,
		Uptime:  time.Since(healthStartedAt).Round(time.Second).String(),
	})

	if err != nil {
		return http.NewStatusResponse(protocol.StatusInternalServerError)
	}

	return response.WithHeader(protocol.HeaderCacheControl, "no-cache")
}

// Name returns the stable handler identifier used in logs.
func (h *HealthHandler) Name() string {
	return "health"
}

// NewHealthHandler constructs a health handler from validated config options.
func NewHealthHandler(options *config.HandlerHealthOptions) Handler {
	return &HealthHandler{code: options.Code}
}
