package iapgo

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	tunnel "github.com/davidspek/go-iap-tunnel/pkg"
)

func StartIapTunnel(ctx context.Context, listener net.Listener,
	conf Config, logger *slog.Logger,
) *tunnel.TunnelManager {
	target := tunnel.TunnelTarget{
		Project:   conf.ProjectID,
		Zone:      conf.Zone,
		Instance:  conf.Instance,
		Port:      conf.RemotePort,
		Interface: conf.RemoteNic,
	}

	// If SSH Tunnelling is used then the target port is always 22
	if conf.SshTunnel != nil {
		target.Port = 22
	}

	logger.Debug("starting IAP Tunnel Manager", "remote port", target.Port)
	tunnelManager := tunnel.NewTunnelManager(target, nil)

	go func() {
		logger.Debug("tunnelManager.Serve() starting to wait for connection")

		err := tunnelManager.Serve(ctx, listener)

		if err != nil {
			logger.Error("tunnelManager.Serve() failed", "err", err)

			return
		}

		logger.Debug("tunnelManager.Serve() exited normally")
	}()

	return tunnelManager
}

func StartIapTunnelMgr(ctx context.Context, iapLsnr net.Listener, cfg *Config, logger *slog.Logger) (*tunnel.TunnelManager, error) {
	tunnelMgr := StartIapTunnel(ctx, iapLsnr, *cfg, logger)

	iapLsnrPort, err := GetPortFromTcpAddr(iapLsnr, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get port from IAP listener: %w", err)
	}

	logger.Debug("iapLsnr is listening on TCP port", "port", iapLsnrPort)

	select {
	case <-time.After(1 * time.Second):
		return nil, fmt.Errorf("timed out waiting for the IAP tunnel to be ready")

	case err := <-tunnelMgr.Errors():
		return nil, fmt.Errorf("IAP tunnel returned an error: %w", err)

	case <-tunnelMgr.Ready():
		logger.Info("IAP tunnel is ready")

		break
	}
	return tunnelMgr, nil
}
