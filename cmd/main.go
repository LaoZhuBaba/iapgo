package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/LaoZhuBaba/iapgo/pkg/iapgo"
)

const (
	defaultConfigFileName = "iapgo.yaml"
	defaultConfigSection  = "default"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	helpPtr := flag.Bool("h", false, "print a usage message")
	configSectionPtr := flag.String(
		"c",
		defaultConfigSection,
		"select a non-default configuration file section",
	)
	configFilePtr := flag.String(
		"f",
		defaultConfigFileName,
		"select a non-default configuration file",
	)
	verbosePtr := flag.Bool("v", false, "print debugging messages")

	flag.Parse()

	if *helpPtr {
		flag.Usage()
		fmt.Printf("\nExample configuration file...\n")
		fmt.Printf("%s\n", iapgo.ExampleConfig)

		return
	}

	var logLevel slog.LevelVar
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: &logLevel,
	}))
	slog.SetDefault(logger)

	if *verbosePtr {
		logLevel.Set(slog.LevelDebug)
	}

	cfg, err := iapgo.GetConfig(ctx, *configFilePtr, *configSectionPtr, logger)
	if err != nil {
		flag.Usage()

		return
	}

	logger.Debug("config", "cfgMap[*configSectionPtr]", *cfg)

	// If SSH tunnelling is being used then localIapPort should be zero.  This means the iapLsnr will
	// use a random ephemeral port.  If SSH tunnelling is not being used then localIapPort should be set
	// to the value of cfg.LocalPort (which will also be zero if value is not configured).
	var localIapPort int

	// Because the listener port we get from the config may be zero we need to check the actual
	// value that RunCmd() uses to set the $IAPGO_LISTEN_POR environment variable.  Also the port
	// that RunCmd needs may be the IAP listener port or the SSH listener port, depending on config.
	var sshLsnrPort, portForRunCmd int

	if cfg.SshTunnel == nil {
		localIapPort = cfg.LocalPort
	}

	// This is the localhost TCP port that connects to the IAP tunnel.
	iapLsnr, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", localIapPort))
	if err != nil {
		logger.Error("failed to listen (iapLsnr)", "error", err)

		return
	}

	defer func() {
		logger.Debug("closing IAP listener")
		_ = iapLsnr.Close()
	}()

	iapLsnrPort, err := iapgo.GetPortFromTcpAddr(iapLsnr, logger)
	if err != nil {
		logger.Error("failed to get port from IAP listener", "error", err)
		return
	}

	logger.Debug("iapLsnr is listening on TCP port", "port", iapLsnrPort)

	tunnelMgr := iapgo.StartIapTunnel(ctx, iapLsnr, *cfg, logger)

	select {
	case <-time.After(1 * time.Second):
		logger.Warn("timed out waiting for the IAP tunnel to be ready")

		return
	case err := <-tunnelMgr.Errors():
		logger.Error("IAP tunnel returned an error", "error", err)

		return
	case <-tunnelMgr.Ready():
		logger.Info("IAP tunnel is ready")

		break
	}

	go func() {
		err := <-tunnelMgr.Errors()
		logger.Error("iap tunnel manager returned an error", "error", err)
	}()

	if cfg.SshTunnel != nil {
		sshLsnr, err := iapgo.StartSshTunnel(ctx, cfg, iapLsnrPort, sshLsnrPort, logger)

		if err != nil {
			logger.Error("failed to start ssh tunnel", "error", err)
			return
		}
		defer func() {
			logger.Debug("closing SSH listener")
			_ = sshLsnr.Close()
		}()
	}

	if cfg.Exec == nil {
		logger.Debug("no Exec command so wait forever.  Enter Control-C to exit")
		<-ctx.Done()
	}

	if cfg.SshTunnel == nil {
		portForRunCmd = iapLsnrPort
	} else {
		portForRunCmd = sshLsnrPort
	}

	iapgo.RunCmd(ctx, cfg.Exec, portForRunCmd, logger)
}
