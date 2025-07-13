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

func NewIapTunnel(cfg *config.Config, server TunnelServer, listener net.Listener, logger *slog.Logger) *IapTunnel {
	return &IapTunnel{
		config:    cfg,
		logger:    logger,
		listener:  listener,
		tunnelMgr: server,
	}
}

func (t *IapTunnel) Errors() <-chan error {
	if t.tunnelMgr == nil {
		return nil
	}

	return t.tunnelMgr.Errors()
}

func (t *IapTunnel) Start(ctx context.Context) error {
	t.startMgr(ctx)

	iapLsnrPort, err := util.GetPortFromTcpAddr(t.listener, t.logger)
	if err != nil {
		return fmt.Errorf("failed to get port from IAP listener: %w", err)
	}

	t.logger.Debug("iapLsnr is listening on TCP port", "port", iapLsnrPort)
	select {
	case <-time.After(1 * time.Second):
		return constants.ErrTunnelReadyTimeout

	case err := <-t.tunnelMgr.Errors():
		return fmt.Errorf("%w: %w", constants.ErrTunnelReturnedError, err)

	case <-t.tunnelMgr.Ready():
		t.logger.Info("IAP tunnel is ready")

		break
	}

	return nil
}

func (t *IapTunnel) startMgr(ctx context.Context) {
	target := tunnel.TunnelTarget{
		Project:   t.config.ProjectID,
		Zone:      t.config.Zone,
		Instance:  t.config.Instance,
		Port:      t.config.RemotePort,
		Interface: t.config.RemoteNic,
	}

	// If SSH Tunnelling is used then the target port is always 22
	if t.config.SshTunnel != nil {
		target.Port = 22
	}

	t.logger.Debug("starting IAP Tunnel Manager", "remote port", target.Port)
	t.tunnelMgr = tunnel.NewTunnelManager(target, nil)

	go func() {
		t.logger.Debug("tunnelManager.Serve() starting to wait for connection")

		err := t.tunnelMgr.Serve(ctx, t.listener)
		if err != nil {
			t.logger.Error("tunnelManager.Serve() failed", "err", err)

			return
		}

		t.logger.Debug("tunnelManager.Serve() exited normally")
	}()
}
