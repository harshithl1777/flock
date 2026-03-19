package server

import (
	"net"
	"strconv"

	"github.com/harshithl1777/flock/internal/config"
	"github.com/harshithl1777/flock/internal/errors"
	"github.com/harshithl1777/flock/internal/logger"
)

type Server struct {
	cfg *config.Config
	ln  net.Listener
}

// New constructs a Server from the provided configuration.
func New(cfg *config.Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

// Start opens the configured TCP listener and serves incoming connections.
//
// It continues accepting connections until listener creation fails or the process exits.
func (srv *Server) Start() error {
	addr := ":" + strconv.Itoa(srv.cfg.Network.Port)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.Wrap("open tcp listener", err)
	}

	srv.ln = ln
	logger.Info("listening on %s", addr)

	for {
		conn, err := srv.ln.Accept()
		if err != nil {
			logger.Error("accept new connection: %v", err)
			continue
		}

		logger.Info("accepted new connection from %s", conn.RemoteAddr().String())
		srv.handleConnection(conn)
	}
}
