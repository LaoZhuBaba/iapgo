package iapgo

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	tunnel "github.com/davidspek/go-iap-tunnel/pkg"
)

func StartIapTunnel(ctx context.Context, conf Config, logger *slog.Logger, portCh chan<- int, errCh chan<- error) {
	localPort := conf.LocalPort

	go func() {
		target := tunnel.TunnelTarget{
			Project:   conf.ProjectID,
			Zone:      conf.Zone,
			Instance:  conf.Instance,
			Port:      conf.RemotePort,
			Interface: conf.RemoteNic,
		}
		if conf.SshTunnelTo == "" {
			target.Port = conf.RemotePort
		} else {
			logger.Debug("connecting IAP tunnel to TCP port 22")
			target.Port = 22
			localPort = 0
		}

		listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", localPort))
		if err != nil {
			errCh <- err
			return
		}
		portCh <- listener.Addr().(*net.TCPAddr).Port

		manager := tunnel.NewTunnelManager(target, nil)

		err = manager.Serve(ctx, listener)
		if err != nil {
			logger.Error("failed to start tunnel", "error", err)
			return
		}
		logger.Debug("after starting IAP server", "port", listener.Addr().(*net.TCPAddr).Port)

	}()
	//<-ctx.Done()
}
