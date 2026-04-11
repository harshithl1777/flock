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

// RoutingDecision classifies how the caller should handle a matched request.
type RoutingDecision uint16

const (
	Forward RoutingDecision = iota
	Options
	NotFound
	MethodNotAllowed
)

// Match captures the best route lookup result along with any routing error.
type Match struct {
	Route    *Route
	Err      *errors.OpError
	Decision RoutingDecision
}

// Resolve returns the best route match together with the caller action needed
// to complete request handling.
func (r *Router) Resolve(method protocol.Method, path string) Match {
	pathMatch, methodMatch := r.match(method, path)

	var decision RoutingDecision
	var route *Route
	var err *errors.OpError

	if method == protocol.Options && pathMatch != nil {
		route = pathMatch
		decision = Options
	} else if methodMatch != nil {
		route = methodMatch
		decision = Forward
	} else if pathMatch != nil {
		route = pathMatch
		decision = MethodNotAllowed
		err = errors.Newf(
			errors.MethodNotAllowedKind,
			"match request route",
			"requested method %s not allowed",
			method,
		)
	} else {
		decision = NotFound
		err = errors.Newf(
			errors.NotFoundKind,
			"match request route",
			"no matching route found for %s %s",
			method,
			path,
		)
	}

	return Match{Route: route, Decision: decision, Err: err}
}

// match returns the longest path match and the longest method-allowed match.
func (r *Router) match(method protocol.Method, path string) (*Route, *Route) {
	path = normalizePath(path)

	var pathMatch *Route
	var methodMatch *Route

	for i := range r.routes {
		route := &r.routes[i]

		if route.Path != path {
			if !strings.HasPrefix(path, route.Path) {
				continue
			}

			isBoundary := len(path) == len(route.Path) || path[len(route.Path)] == '/'
			if !isBoundary {
				continue
			}
		}

		if pathMatch == nil || len(route.Path) > len(pathMatch.Path) {
			pathMatch = route
		}

		if !route.Methods.Allows(method) {
			continue
		}

		if methodMatch == nil || len(route.Path) > len(methodMatch.Path) {
			methodMatch = route
		}
	}

	return pathMatch, methodMatch
}

// New builds the runtime router from the validated configuration routes.
func New(routes config.RoutesConfig) *Router {
	builtRoutes := make([]Route, 0, len(routes))

	for _, route := range routes {
		methods := newMethodMask(route.Methods)
		builtRoutes = append(builtRoutes, Route{
			Path:        normalizePath(route.Path),
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

	panic("newHandler: route has no handler options configured")
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
