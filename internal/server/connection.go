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
	"go.uber.org/zap"
)

type Connection struct {
	net.Conn
	log        *zap.Logger
	remoteAddr string
}

type RequestContext struct {
	id      string
	startTs time.Time
	log     *zap.Logger
}

// serve reads a single HTTP request, writes a response, and closes the connection.
func (c *Connection) serve() {
	defer func() {
		c.Conn.Close()
		c.log.Info("close connection")
	}()

	c.log.Info("accept connection")

	ctx := newRequestContext()
	ctx.log.Info("serve new request")

	reader := bufio.NewReader(c.Conn)
	request, err := http.ReadRequest(reader)

	if err != nil {
		c.fail(ctx, err)
		return
	}

	ctx.log.Info(
		"route request",
		logger.String("method", string(request.Method)),
		logger.String("path", request.Path),
	)

	r := http.NewStatusResponse(http.StatusOK)
	c.write(ctx, r, nil)
}

// fail converts an operational error into an HTTP error response and writes it.
func (c *Connection) fail(ctx *RequestContext, err *errors.OpError) {
	r, marshalErr := http.NewErrorResponse(err)
	if marshalErr != nil {
		r = http.NewStatusResponse(http.StatusInternalServerError)
		err = marshalErr
	}

	c.write(ctx, r, err)
}

// write finalizes standard response headers, writes the response, and logs the result.
func (c *Connection) write(ctx *RequestContext, r *http.Response, err *errors.OpError) {
	r = r.WithHeader(http.HeaderServer, FullServerVersion()).
		WithHeader(http.HeaderXRequestId, ctx.id).
		WithHeader(http.HeaderConnection, "close")

	_, writeErr := r.WriteTo(c.Conn)

	latency := time.Since(ctx.startTs)

	log := ctx.log.With(
		logger.Int("status", r.StatusCode),
		logger.Duration("latency", latency),
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

// newConnection wraps a net.Conn with server logging metadata.
func newConnection(netConn net.Conn) *Connection {
	remoteAddr := netConn.RemoteAddr().String()
	return &Connection{
		Conn:       netConn,
		log:        logger.With(logger.String("remote", remoteAddr)),
		remoteAddr: netConn.RemoteAddr().String(),
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
	return fmt.Sprintf("req_%d_%d", ts.UnixMilli(), n)
}
