package server

import (
	"bufio"
	"net"

	"github.com/harshithl1777/flock/core/httpcore"
	"github.com/harshithl1777/flock/core/utils/errors"
	"github.com/harshithl1777/flock/core/utils/logger"
)

// readRequestLines reads and logs request lines from reader.
//
// It stops after the empty line that terminates the HTTP header section.
func readRequestLines(reader *bufio.Reader) error {
	for i := 1; ; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			return errors.Wrap("read request lines", err)
		}

		logger.Info("request line %d: %q", i, line)

		if line == "\r\n" {
			break
		}
	}

	return nil
}

// handleConnection reads a single HTTP request from conn.
//
// It logs the request lines, writes a plain-text response, and closes the connection before returning.
func (srv *Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		logger.Info("closed connection from: %s", conn.RemoteAddr().String())
	}()

	reader := bufio.NewReader(conn)
	if err := readRequestLines(reader); err != nil {
		logger.Error("handle connection: %v", err)
		return
	}
	response := httpcore.NewResponse("Hello World!") // TODO: change
	responseBytes := response.SerializeResponse()

	if _, err := conn.Write(responseBytes); err != nil {
		logger.Error("write response to connection: %v", err)
	} else {
		logger.Info("wrote response to connection from: %s", conn.RemoteAddr().String())
	}
}
