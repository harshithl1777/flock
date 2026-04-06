package server

import (
	"bufio"
	"fmt"
	"net"
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
	router *router.Router
	log    *zap.Logger
	remote string
}

type RequestContext struct {
	id      string
	startTs time.Time
	log     *zap.Logger
}

// serve reads one request, resolves it through the router, writes the response,
// and closes the connection.
func (c *Connection) serve() {
	defer func() {
		c.Close()
		c.log.Info("close connection")
	}()

	c.log.Info("accept connection")

	ctx := newRequestContext()
	ctx.log.Info("serve new request")

	reader := bufio.NewReader(c)
	req, err := http.ReadRequest(reader)

	if err != nil {
		c.fail(ctx, err)
		return
	}

	handler, errResp := c.router.Resolve(req.Method, req.Path)
	if errResp != nil {
		c.write(ctx, errResp, nil)
		return
	}

	if handler == nil {
		c.write(ctx, http.NewStatusResponse(protocol.StatusInternalServerError), nil)
		return
	}

	ctx.log.Info(
		"route request",
		logger.String("method", string(req.Method)),
		logger.String("path", req.Path),
		logger.String("handler", handler.Name()),
	)

	resp := handler.Handle(req)
	c.write(ctx, resp, nil)
}

// fail converts an operational error into an HTTP error response and writes it.
func (c *Connection) fail(ctx *RequestContext, err *errors.OpError) {
	r, marshalErr := http.NewErrorResponse(err)
	if marshalErr != nil {
		r = http.NewStatusResponse(protocol.StatusInternalServerError)
		err = marshalErr
	}

	c.write(ctx, r, err)
}

// write attaches the standard server headers, writes the response to the
// connection, and records the result in the request log.
func (c *Connection) write(ctx *RequestContext, r *http.Response, err *errors.OpError) {
	r = r.WithHeader(protocol.HeaderServer, protocol.FullServerVersion()).
		WithHeader(protocol.HeaderXRequestId, ctx.id).
		WithHeader(protocol.HeaderConnection, "close")

	_, writeErr := r.WriteTo(c)

	log := ctx.log.With(
		logger.Int("status", r.StatusCode),
		logger.Duration("latency", time.Since(ctx.startTs)),
	)

	if writeErr != nil {
		c.log.Error("failed to write response", logger.Err(writeErr))
		return
	}

	if err != nil {
		log.Info(
			"send error response",
			logger.Err(err),
		)
	} else {
		log.Info("send success response")
	}
}

// NewConnection wraps a net.Conn with server logging metadata.
func NewConnection(netConn net.Conn, router *router.Router) *Connection {
	remoteAddr := netConn.RemoteAddr().String()
	return &Connection{
		Conn:   netConn,
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
