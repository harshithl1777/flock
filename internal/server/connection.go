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
	defer func() {
		conn.Close()
		logger.Info("closed connection from: %s", conn.RemoteAddr().String())
	}()

	reader := bufio.NewReader(conn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		logger.Error("handle connection: %v", err)
		return
	} else {
		logger.Info("read request as: %v", request)
	}

	response := http.NewResponse(http.StatusOK, "Hello World!") // TODO: change

	if _, err := response.WriteTo(conn); err != nil {
		logger.Error("write response to connection: %v", err)
	} else {
		logger.Info("wrote response to connection from: %s", conn.RemoteAddr().String())
	}
}
