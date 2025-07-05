package ssh

import (
	"context"
	"io"
	"log/slog"
	"net"
	"sync"
)

type Handler struct {
	logger     *slog.Logger
	localConn  net.Conn
	tunnelConn net.Conn
}

func NewHandler(localConn net.Conn, tunnelConn net.Conn, logger *slog.Logger) *Handler {
	return &Handler{
		logger:     logger,
		localConn:  localConn,
		tunnelConn: tunnelConn,
	}
}

func (h *Handler) Handle(ctx context.Context) {
	var wait sync.WaitGroup

	h.logger.Debug("started handling tunnel i/o")

	wait.Add(1)

	go func() {
		defer wait.Done()

		_, err := io.Copy(h.tunnelConn, h.localConn)
		if err != nil {
			h.logger.Error("failed to copy local connection", "error", err)

			return
		}

		h.logger.Debug("io.Copy local connection completed")
	}()

	wait.Add(1)

	go func() {
		defer wait.Done()

		_, err := io.Copy(h.localConn, h.tunnelConn)
		if err != nil {
			h.logger.Error("failed to copy tunnel connection", "error", err)
		}

		h.logger.Debug("io.Copy tunnel connection completed")
	}()

	wait.Wait()

	_ = h.localConn.Close()
	_ = h.tunnelConn.Close()
	h.logger.Debug("Handler exiting")
}
