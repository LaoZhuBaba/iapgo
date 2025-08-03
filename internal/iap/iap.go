package iap

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	config "github.com/LaoZhuBaba/iapgo/v2/internal/config"
	"github.com/LaoZhuBaba/iapgo/v2/internal/constants"
	"github.com/LaoZhuBaba/iapgo/v2/internal/util"
	tunnel "github.com/davidspek/go-iap-tunnel/pkg"
)

type TunnelServer interface {
	Serve(ctx context.Context, lis net.Listener) error
	Errors() <-chan error
	Ready() <-chan struct{}
}
type IapTunnel struct {
	config    *config.Config
	listener  net.Listener
	logger    *slog.Logger
	tunnelMgr TunnelServer
}

func NewIapTunnel(cfg *config.Config, listener net.Listener, logger *slog.Logger) (*IapTunnel, error) {
	if cfg == nil || logger == nil || listener == nil {
		return nil, constants.ErrNilParameter
	}
	target := tunnel.TunnelTarget{
		Project:   cfg.ProjectID,
		Zone:      cfg.Zone,
		Instance:  cfg.Instance,
		Port:      cfg.RemotePort,
		Interface: cfg.RemoteNic,
	}

	// If SSH Tunnelling is used then the target port is always 22
	if cfg.SshTunnel != nil {
		target.Port = 22
	}

	logger.Debug("starting IAP Tunnel Manager", "remote port", target.Port)

	tunnelMgr := tunnel.NewTunnelManager(target, nil)
	return &IapTunnel{
		config:    cfg,
		logger:    logger,
		listener:  listener,
		tunnelMgr: tunnelMgr,
	}, nil
}

func (t *IapTunnel) Errors() <-chan error {
	if t.tunnelMgr == nil {
		return nil
	}

	return t.tunnelMgr.Errors()
}

func (t *IapTunnel) Start(ctx context.Context) error {
	go t.startMgr(ctx)

	iapLsnrPort, err := util.GetPortFromTcpAddr(t.listener, t.logger)
	if err != nil {
		return fmt.Errorf("failed to get port from IAP listener: %w", err)
	}

	t.logger.Debug("iapLsnr is listening on TCP port", "port", iapLsnrPort)

	errCh := t.tunnelMgr.Errors()
	readyCh := t.tunnelMgr.Ready()

	select {
	case <-time.After(1 * time.Second):
		return constants.ErrTunnelReadyTimeout

	case err := <-errCh:
		return fmt.Errorf("%w: %w", constants.ErrTunnelReturnedError, err)

	case ready := <-readyCh:
		t.logger.Info("IAP tunnel is ready", "ready", ready)

		break
	}

	return nil
}

func (t *IapTunnel) startMgr(ctx context.Context) {
	t.logger.Debug("tunnelManager.Serve() starting to wait for connection")

	err := t.tunnelMgr.Serve(ctx, t.listener)
	if err != nil {
		t.logger.Error("tunnelManager.Serve() failed", "err", err)

		return
	}

	t.logger.Debug("tunnelManager.Serve() exited normally")
}
