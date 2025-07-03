package iapgo

import (
	"context"
	"log/slog"
	"net"

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
