package server

import (
	"bufio"
	"math"
	"net"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/errors"
	"github.com/harshithl1777/flock/internal/http"
	"github.com/harshithl1777/flock/internal/logger"
	"github.com/harshithl1777/flock/internal/protocol"
	"github.com/harshithl1777/flock/internal/router"
	"go.uber.org/zap"
)

type Connection struct {
	net.Conn
	cfg            *config.Config
	reader         *bufio.Reader
	router         *router.Router
	log            *zap.Logger
	ctx            ConnectionContext
	remote         string
	requestsServed int64
	bytesWritten   int64
}

type ConnectionContext struct {
	log     *zap.Logger
	id      uint64
	startTs time.Time
}
type RequestContext struct {
	log       *zap.Logger
	id        uint64
	startTs   time.Time
	version   protocol.Version
	keepAlive bool
}

func (c *Connection) init() {
	if c.reader == nil {
		c.reader = bufio.NewReader(c.Conn)
	}
}

func (c *Connection) serve() {
	c.init()
	c.ctx.log.Debug("accept connection")

	defer func() {
		recovered := recover()
		if recovered == nil {
			c.Close()
			c.ctx.log.Debug("close connection")
			return
		}

		c.ctx.log.Error(
			"connection panic",
			logger.Any("panic", recovered),
			logger.ByteString("stack", debug.Stack()),
		)

		if c.bytesWritten == 0 {
			c.panicWrite(c.newPanicContext())
		}

		c.Close()
		c.ctx.log.Debug("close connection after panic")
	}()

	for {
		close := c.handle()
		if close {
			return
		}
	}

}

// handle reads one request, resolves it through the router, writes the response,
// and closes the connection.
func (c *Connection) handle() (close bool) {
	ctx := c.newRequestContext()
	ctx.log.Debug("serve new request")

	c.setReadDeadlineForNextRequest()

	req, err := http.ReadRequest(c.reader)
	if err != nil {
		if err.Kind == errors.ClientClosedConnectionKind {
			ctx.log.Debug("client close connection")
			return true
		}
		if err.Kind == errors.RequestTimeoutKind && c.requestsServed > 0 {
			ctx.log.Debug("idle timeout")
			return true
		}

		if writeErr := c.failWrite(ctx, err); writeErr != nil {
			return true
		}

		c.increment()
		return !ctx.keepAlive
	}

	c.clearReadDeadline()

	ctx.keepAlive = shouldKeepAlive(req)
	ctx.version = req.Version

	if ctx.keepAlive && c.requestsServed+1 >= c.cfg.Network.MaxRequestsPerConnection {
		ctx.keepAlive = false
	}

	match := c.router.Resolve(req.Method, req.Path)
	if match.Decision != router.Forward {
		if writeErr := c.routerWrite(ctx, match); writeErr != nil {
			return true
		}
		c.increment()
		return !ctx.keepAlive
	}

	ctx.log.Debug(
		"route request",
		logger.String("method", string(req.Method)),
		logger.String("path", req.Path),
		logger.String("handler", match.Route.Handler.Name()),
	)

	resp := match.Route.Handler.Handle(req)
	if writeErr := c.successWrite(ctx, resp); writeErr != nil {
		return true
	}

	c.increment()
	return !ctx.keepAlive
}

// successWrite records a successful handler response with no associated error.
func (c *Connection) successWrite(ctx RequestContext, r *http.Response) *errors.OpError {
	return c.write(ctx, r, nil)
}

// routerWrite converts a router-owned decision into its HTTP response.
func (c *Connection) routerWrite(ctx RequestContext, match router.Match) *errors.OpError {
	var r *http.Response
	var allow string
	err := match.Err

	if match.Route != nil {
		allow = match.Route.AllowHeader
	}

	switch match.Decision {
	case router.Options:
		r = http.NewStatusResponse(protocol.StatusNoContent).
			WithHeader(protocol.HeaderAllow, allow)
	case router.MethodNotAllowed:
		errResp, marshalErr := http.NewErrorResponse(err)
		if marshalErr != nil {
			r = http.NewStatusResponse(protocol.StatusInternalServerError)
			err = marshalErr
		} else {
			r = errResp.WithHeader(protocol.HeaderAllow, allow)
		}
	default:
		errResp, marshalErr := http.NewErrorResponse(match.Err)
		if marshalErr != nil {
			r = http.NewStatusResponse(protocol.StatusInternalServerError)
			err = marshalErr
		} else {
			r = errResp
		}
	}

	return c.write(ctx, r, err)
}

// failWrite converts an operational error into an HTTP error response and writes it.
func (c *Connection) failWrite(ctx RequestContext, err *errors.OpError) *errors.OpError {
	r, marshalErr := http.NewErrorResponse(err)
	if marshalErr != nil {
		r = http.NewStatusResponse(protocol.StatusInternalServerError)
		err = marshalErr
	}

	return c.write(ctx, r, err)
}

// panicWrite emits a generic 500 response after a panic that occurred before
// any bytes were written to the connection.
func (c *Connection) panicWrite(ctx RequestContext) {
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
func (c *Connection) write(ctx RequestContext, r *http.Response, err *errors.OpError) *errors.OpError {
	r = r.WithHeader(protocol.HeaderServer, protocol.FullServerVersion()).
		WithHeader(protocol.HeaderXRequestId, strconv.FormatUint(ctx.id, 10))

	if r.StatusCode == int(protocol.StatusNoContent) || r.StatusCode == int(protocol.StatusNotModified) {
		r = r.WithBody("")
	}

	if ctx.version == protocol.HTTP10 && ctx.keepAlive {
		r = r.WithHeader(protocol.HeaderConnection, "keep-alive")
	} else if !ctx.keepAlive {
		r = r.WithHeader(protocol.HeaderConnection, "close")
	}

	n, writeErr := r.WriteTo(c)
	c.bytesWritten += n

	log := ctx.log.With(
		logger.Int("status", r.StatusCode),
		logger.Duration("latency", time.Since(ctx.startTs)),
	)

	if writeErr != nil {
		log.Warn("failed to write response", logger.Err(writeErr))
		return errors.Wrap(errors.ConnectionWriteKind, "write response", writeErr)
	}

	if err == nil {
		log.Info("send response")
	} else {
		log.Info("send error response", logger.Err(err))
	}
	return nil
}

func (c *Connection) increment() {
	c.requestsServed++
}

func (c *Connection) setReadDeadlineForNextRequest() {
	now := time.Now()

	if c.requestsServed == 0 {
		_ = c.SetReadDeadline(now.Add(c.cfg.Timeouts.Read))
		return
	}

	_ = c.SetReadDeadline(now.Add(c.cfg.Timeouts.Idle))
}

func (c *Connection) clearReadDeadline() {
	c.SetReadDeadline(time.Time{})
}

// newRequestContext allocates per-request logging and timing metadata.
func (c *Connection) newRequestContext() RequestContext {
	ts := time.Now()
	requestId := newRequestId()
	return RequestContext{
		id:      requestId,
		startTs: ts,
		log:     c.ctx.log.With(logger.Uint64("request_id", requestId)),
	}
}

func (c *Connection) newPanicContext() RequestContext {
	ts := time.Now()
	var panicId uint64 = math.MaxUint64
	return RequestContext{
		id:        panicId,
		startTs:   ts,
		log:       c.ctx.log.With(logger.Bool("panic_context", true)),
		keepAlive: false,
	}
}

// NewConnection wraps a net.Conn with server logging metadata.
func NewConnection(netConn net.Conn, cfg *config.Config, router *router.Router) *Connection {
	remoteAddr := netConn.RemoteAddr().String()
	return &Connection{
		Conn:   netConn,
		cfg:    cfg,
		reader: bufio.NewReader(netConn),
		router: router,
		ctx:    newConnectionContext(),
		remote: remoteAddr,
	}
}

func shouldKeepAlive(req *http.Request) bool {
	connectionValue := req.Headers[string(protocol.HeaderConnection)]

	switch req.Version {
	case protocol.HTTP11:
		return !strings.EqualFold(connectionValue, "close")

	case protocol.HTTP10:
		return strings.EqualFold(connectionValue, "keep-alive")

	default:
		return false
	}
}

// newConnectionContext allocates per-request logging and timing metadata.
func newConnectionContext() ConnectionContext {
	ts := time.Now()
	connId := newConnectionId()
	return ConnectionContext{
		id:      connId,
		startTs: ts,
		log:     logger.With(logger.Uint64("connection_id", connId)),
	}
}

var connectionSequence atomic.Uint64

// newRequestId returns a monotonically increasing request identifier with a time prefix.
func newConnectionId() uint64 {
	n := connectionSequence.Add(1)
	return n
}

var requestSequence atomic.Uint64

// newRequestId returns a monotonically increasing request identifier with a time prefix.
func newRequestId() uint64 {
	n := requestSequence.Add(1)
	return n
}
