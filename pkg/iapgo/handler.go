package iapgo

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

	h.logger.Debug("in HandleClientConnection, created tunnel connection")

	wait.Add(1)
	go func(ctx context.Context) {
		defer wait.Done()
		_, err := io.Copy(h.tunnelConn, h.localConn)
		if err != nil {
			//h.errCh <- fmt.Errorf("in HandleClientConnection: failed to copy local connection: %w", err)
			h.logger.Error("failed to copy local connection", "error", err)
		}
		h.logger.Debug("io.Copy local connection completed")

	}(ctx)

	wait.Add(1)
	go func(ctx context.Context) {
		defer wait.Done()
		_, err := io.Copy(h.localConn, h.tunnelConn)
		if err != nil {
			//h.errCh <- fmt.Errorf("in HandleClientConnection: failed to copy tunnel connection: %w", err)
			h.logger.Error("failed to copy tunnel connection", "error", err)
		}
		h.logger.Debug("io.Copy tunnel connection completed")
	}(ctx)

	wait.Wait()
	_ = h.localConn.Close()
	_ = h.tunnelConn.Close()
	h.logger.Debug("Handler exiting")
}
