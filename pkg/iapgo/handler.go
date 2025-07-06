package iapgo

import (
	"context"
	"errors"
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

func (h *Handler) Handle(ctx context.Context) {
	ctx, cancel := context.WithCancelCause(ctx)

	h.logger.Debug("started handling tunnel i/o")

	go func() {
		_, err := io.Copy(h.tunnelConn, h.localConn)
		cancel(err)
	}()

	go func() {
		_, err := io.Copy(h.localConn, h.tunnelConn)
		cancel(err)
	}()

	<-ctx.Done()
	if errors.Is(ctx.Err(), context.Canceled) {
		cause := context.Cause(ctx)
		if cause != nil {
			h.logger.Error("error in I/O handler", "error", cause)
		}
	}
	_ = h.localConn.Close()
	_ = h.tunnelConn.Close()
	h.logger.Debug("Handler exiting")
}
