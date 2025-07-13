package ssh

import (
	"io"
	"log/slog"
)

type Handler struct {
	logger     *slog.Logger
	localConn  io.ReadWriteCloser
	tunnelConn io.ReadWriteCloser
}

func NewHandler(localConn io.ReadWriteCloser, tunnelConn io.ReadWriteCloser, logger *slog.Logger) *Handler {
	return &Handler{
		logger:     logger,
		localConn:  localConn,
		tunnelConn: tunnelConn,
	}
}

func (h *Handler) Handle() (error, error) {
	localConnCh := make(chan error)
	tunnelConnCh := make(chan error)
	h.logger.Debug("started handling tunnel i/o")
	go func() {
		_, err := io.Copy(h.tunnelConn, h.localConn)
		localConnCh <- err
		h.logger.Debug("io.Copy local connection completed")
	}()

	go func() {
		_, err := io.Copy(h.localConn, h.tunnelConn)
		tunnelConnCh <- err
		h.logger.Debug("io.Copy tunnel connection completed")
	}()

	localConnErr := <-localConnCh
	tunnelConnErr := <-tunnelConnCh
	_ = h.localConn.Close()
	_ = h.tunnelConn.Close()
	return localConnErr, tunnelConnErr
}
