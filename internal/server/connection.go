package server

import (
	"bufio"
	"net"

	"github.com/harshithl1777/flock/internal/http"
	"github.com/harshithl1777/flock/internal/logger"
)

// handleConnection reads a single HTTP request from conn.
//
// It logs the request lines, writes a plain-text response, and closes the connection before returning.
func (srv *Server) handleConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	log := logger.With(logger.String("remote_addr", remoteAddr))
	defer func() {
		conn.Close()
		log.Info("closed connection")
	}()

	reader := bufio.NewReader(conn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		log.Error("failed to handle connection", logger.Err(err))
		return
	}

	log.Info(
		"accept connection",
		logger.String("method", string(request.Method)),
		logger.String("path", request.Path),
		logger.String("version", string(request.Version)),
	)

	response := http.NewResponse(http.StatusOK, "Hello World!") // TODO: change

	n, err := response.WriteTo(conn)
	if err != nil {
		log.Error("failed to write response", logger.Err(err))
	} else {
		log.Info(
			"wrote response",
			logger.Int("bytes", int(n)),
			logger.Int("status_code", response.StatusCode),
		)
	}
}
