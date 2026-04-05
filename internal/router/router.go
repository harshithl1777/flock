package router

import (
	"strings"

	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/errors"
	"github.com/harshithl1777/flock/internal/handler"
	"github.com/harshithl1777/flock/internal/http"
	"github.com/harshithl1777/flock/internal/protocol"
)

// Route is the runtime representation of a configured route.
type Route struct {
	Path        string
	Methods     MethodMask
	AllowHeader string
	Handler     handler.Handler
}

type Router struct {
	routes []Route
}

// Resolve returns either a ready-to-write response for router-owned outcomes
// or the matched handler for normal request dispatch.
func (r *Router) Resolve(method protocol.Method, path string) (*http.Response, handler.Handler) {
	path = normalizePath(path)

	var bestPathMatch *Route
	var bestAllowedMatch *Route

	for i := range r.routes {
		route := &r.routes[i]
		routePath := normalizePath(route.Path)

		if routePath != path {
			if !strings.HasPrefix(path, routePath) {
				continue
			}

			isBoundary := len(path) == len(routePath) || path[len(routePath)] == '/'
			if !isBoundary {
				continue
			}
		}

		if bestPathMatch == nil || len(routePath) > len(bestPathMatch.Path) {
			bestPathMatch = route
		}

		if !route.Methods.Allows(method) {
			continue
		}

		if bestAllowedMatch == nil || len(routePath) > len(bestAllowedMatch.Path) {
			bestAllowedMatch = route
		}
	}

	if method == protocol.Options && bestPathMatch != nil {
		return http.NewStatusResponse(protocol.StatusNoContent).
			WithHeader(protocol.HeaderAllow, bestPathMatch.AllowHeader), nil
	}

	if bestAllowedMatch != nil {
		return nil, bestAllowedMatch.Handler
	}

	if bestPathMatch == nil {
		return newErrorResponse(errors.Newf(errors.NotFoundKind, "match request route", "no matching route found for %s %s", method, path)), nil
	}

	return newErrorResponse(
		errors.Newf(errors.MethodNotAllowedKind, "match request route", "requested method %s not allowed", method),
	).WithHeader(protocol.HeaderAllow, bestPathMatch.AllowHeader), nil
}

// New builds the runtime router from the validated configuration routes.
func New(routes config.RoutesConfig) *Router {
	builtRoutes := make([]Route, 0, len(routes))

	for _, route := range routes {
		methods := newMethodMask(route.Methods)
		builtRoutes = append(builtRoutes, Route{
			Path:        route.Path,
			Methods:     methods,
			AllowHeader: methods.AllowHeader(),
			Handler:     newHandler(route),
		})
	}

	return &Router{
		routes: builtRoutes,
	}
}

// newHandler selects the concrete handler implementation for a configured route.
func newHandler(route config.RouteConfig) handler.Handler {
	if route.HealthOptions != nil {
		return handler.NewHealthHandler(route.HealthOptions)
	} else if route.StatusOptions != nil {
		return handler.NewStatusHandler(route.StatusOptions)
	} else if route.RedirectOptions != nil {
		return handler.NewRedirectHandler(route.RedirectOptions)
	}
	return nil
}

// newErrorResponse builds a router-owned error response and falls back to a
// generic 500 response if JSON serialization fails.
func newErrorResponse(err *errors.OpError) *http.Response {
	resp, marshalErr := http.NewErrorResponse(err)
	if marshalErr != nil {
		return http.NewStatusResponse(protocol.StatusInternalServerError)
	}
	return resp
}

// normalizePath trims a trailing slash from non-root paths so route matching
// treats "/users" and "/users/" as the same location.
func normalizePath(path string) string {
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		return strings.TrimSuffix(path, "/")
	}
	return path
}
