package server

import (
	"net"
	"strconv"

	"github.com/harshithl1777/flock/core/config"
	"github.com/harshithl1777/flock/core/utils/errors"
	"github.com/harshithl1777/flock/core/utils/logger"
)

type Server struct {
	cfg *config.Config
	ln  net.Listener
}

func New(cfg *config.Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

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
