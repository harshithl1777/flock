package server

import (
	"net"
	"strconv"

	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/errors"
	"github.com/harshithl1777/flock/internal/logger"
	"github.com/harshithl1777/flock/internal/router"
)

type Server struct {
	cfg    *config.Config
	ln     net.Listener
	router *router.Router
}

// New constructs a Server from the provided configuration.
func New(cfg *config.Config) *Server {
	return &Server{
		cfg:    cfg,
		router: router.New(cfg.Routes),
	}
}

// Start opens the configured TCP listener and serves incoming connections.
//
// It continues accepting connections until listener creation fails or the process exits.
func (srv *Server) Start() *errors.OpError {
	addr := ":" + strconv.Itoa(srv.cfg.Network.Port)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.Wrap(errors.ServerStartupKind, "open tcp listener", err)
	}

	srv.ln = ln
	logger.Info("listen on tcp", logger.String("address", addr))

	for {
		netConn, err := srv.ln.Accept()
		if err != nil {
			logger.Error("accept new connection", logger.Err(err))
			continue
		}

		conn := NewConnection(netConn, srv.router)
		conn.serve()
	}
}
