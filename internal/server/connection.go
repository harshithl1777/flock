package server

import (
	"bufio"
	"fmt"
	"net"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/harshithl1777/flock/internal/errors"
	"github.com/harshithl1777/flock/internal/http"
	"github.com/harshithl1777/flock/internal/logger"
	"github.com/harshithl1777/flock/internal/protocol"
	"github.com/harshithl1777/flock/internal/router"
	"go.uber.org/zap"
)

type Connection struct {
	net.Conn
	reader       *bufio.Reader
	router       *router.Router
	log          *zap.Logger
	remote       string
	bytesWritten int64
}

type RequestContext struct {
	log     *zap.Logger
	id      string
	startTs time.Time
}

// serve reads one request, resolves it through the router, writes the response,
// and closes the connection.
func (c *Connection) serve() {
	ctx := newRequestContext()
	c.log.Info("accept connection")

	defer func() {
		recovered := recover()
		if recovered == nil {
			c.Close()
			c.log.Info("close connection")
			return
		}

		ctx.log.Error(
			"connection panic",
			logger.Any("panic", recovered),
			logger.ByteString("stack", debug.Stack()),
		)

		if c.bytesWritten == 0 {
			c.panicWrite(ctx)
		}

		c.Close()
		c.log.Info("close connection after panic")
	}()

	ctx.log.Info("serve new request")

	req, err := http.ReadRequest(c.reader)
	if err != nil {
		c.failWrite(ctx, err)
		return
	}

	match := c.router.Resolve(req.Method, req.Path)
	if match.Decision != router.Forward {
		c.routerWrite(ctx, match)
		return
	}

	ctx.log.Info(
		"route request",
		logger.String("method", string(req.Method)),
		logger.String("path", req.Path),
		logger.String("handler", match.Route.Handler.Name()),
	)

	resp := match.Route.Handler.Handle(req)
	c.successWrite(ctx, resp)
}

// successWrite records a successful handler response with no associated error.
func (c *Connection) successWrite(ctx *RequestContext, r *http.Response) {
	c.write(ctx, r, nil)
}

// routerWrite converts a router-owned decision into its HTTP response.
func (c *Connection) routerWrite(ctx *RequestContext, match router.Match) {
	var r *http.Response
	var allow string

	if match.Route != nil {
		allow = match.Route.AllowHeader
	}

	switch match.Decision {
	case router.Options:
		r = http.NewStatusResponse(protocol.StatusNoContent).
			WithHeader(protocol.HeaderAllow, allow)
	case router.MethodNotAllowed:
		r = http.NewStatusResponse(protocol.StatusMethodNotAllowed).
			WithHeader(protocol.HeaderAllow, allow)
	default:
		r = http.NewStatusResponse(protocol.StatusNotFound)
	}

	c.write(ctx, r, match.Err)
}

// failWrite converts an operational error into an HTTP error response and writes it.
func (c *Connection) failWrite(ctx *RequestContext, err *errors.OpError) {
	r, marshalErr := http.NewErrorResponse(err)
	if marshalErr != nil {
		r = http.NewStatusResponse(protocol.StatusInternalServerError)
		err = marshalErr
	}

	c.write(ctx, r, err)
}

// panicWrite emits a generic 500 response after a panic that occurred before
// any bytes were written to the connection.
func (c *Connection) panicWrite(ctx *RequestContext) {
	resp := http.NewStatusResponse(protocol.StatusInternalServerError)
	c.write(
		ctx,
		resp,
		errors.New(
			errors.InternalServerErrorKind,
			"connection",
			"encountered connection panic before write",
		),
	)
}

// write attaches the standard server headers, writes the response to the
// connection, and records the result in the request log.
func (c *Connection) write(ctx *RequestContext, r *http.Response, err *errors.OpError) {
	r = r.WithHeader(protocol.HeaderServer, protocol.FullServerVersion()).
		WithHeader(protocol.HeaderXRequestId, ctx.id).
		WithHeader(protocol.HeaderConnection, "close")

	n, writeErr := r.WriteTo(c)
	c.bytesWritten += n

	log := ctx.log.With(
		logger.Int("status", r.StatusCode),
		logger.Duration("latency", time.Since(ctx.startTs)),
	)

	if writeErr != nil {
		c.log.Error("failed to write response", logger.Err(writeErr))
		return
	}

	if err == nil {
		log.Info("send response")
	} else {
		log.Info("send error response", logger.Err(err))
	}
}

// NewConnection wraps a net.Conn with server logging metadata.
func NewConnection(netConn net.Conn, router *router.Router) *Connection {
	remoteAddr := netConn.RemoteAddr().String()
	return &Connection{
		Conn:   netConn,
		reader: bufio.NewReader(netConn),
		router: router,
		log:    logger.With(logger.String("remote", remoteAddr)),
		remote: remoteAddr,
	}
}

// newRequestContext allocates per-request logging and timing metadata.
func newRequestContext() *RequestContext {
	ts := time.Now()
	requestId := newRequestId(ts)
	return &RequestContext{
		id:      requestId,
		startTs: ts,
		log:     logger.With(logger.String("request_id", requestId)),
	}
}

var requestSequence atomic.Uint64

// newRequestId returns a monotonically increasing request identifier with a time prefix.
func newRequestId(ts time.Time) string {
	n := requestSequence.Add(1)
	return fmt.Sprintf("r_%d_%d", ts.UnixMilli(), n)
}
