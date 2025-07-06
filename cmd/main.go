package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"

<<<<<<< Updated upstream
	"github.com/LaoZhuBaba/iapgo/v2/pkg/iapgo"
=======
	"github.com/LaoZhuBaba/iapgo/v2/internal/config"
	"github.com/LaoZhuBaba/iapgo/v2/internal/constants"
	"github.com/LaoZhuBaba/iapgo/v2/internal/exec"
	"github.com/LaoZhuBaba/iapgo/v2/internal/iap"
	ssh2 "github.com/LaoZhuBaba/iapgo/v2/internal/ssh"
	"github.com/LaoZhuBaba/iapgo/v2/internal/utils"
>>>>>>> Stashed changes
	"golang.org/x/crypto/ssh"
)

const (
	defaultConfigFileName = "iapgo.yaml"
	defaultConfigSection  = "default"
)

type args struct {
	configFile    string
	configSection string
	verbose       bool
}

func getArgs() *args {
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

		return nil
	}
	return &args{
		configFile:    *configFilePtr,
		configSection: *configSectionPtr,
		verbose:       *verbosePtr,
	}
}

func main() {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	args := getArgs()
	if args == nil {
		// This means that the -h flag was passed.
		return
	}

	var logLevel slog.LevelVar
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: &logLevel,
	}))
	slog.SetDefault(logger)

	if args.verbose {
		logLevel.Set(slog.LevelDebug)
	}

<<<<<<< Updated upstream
	cfg, err := iapgo.GetConfig(ctx, *configFilePtr, *configSectionPtr, logger)
=======
	cfg, err := config.GetConfig(ctx, args.configFile, args.configSection, logger)

>>>>>>> Stashed changes
	if err != nil {
		logger.Debug("failed to load configuration", "error", err)

		return
	}

	logger.Debug("config", "cfgMap[*configSectionPtr]", *cfg)

	// If SSH tunnelling is being used then localIapPort should be zero.  This means the iapLsnr will
	// use a random ephemeral port.  If SSH tunnelling is not being used then localIapPort should be set
	// to the value of cfg.LocalPort (which will also be zero if value is not configured).
	var localIapPort int

	// Because the listener port we get from the config may be zero we need to check the actual
	// value that RunCmd() uses to set the $IAPGO_LISTEN_POR environment variable.  Also, the port
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

	tunnel := iapgo.NewIapTunnel(cfg, iapLsnr, logger)

	err = tunnel.Start(ctx)
	if err != nil {
		logger.Error("failed to start tunnel manager", "error", err)

		return
	}

	// Pick up any errors from tunnelMgr, log these and cancel the context.
	go func() {
		ch := tunnel.Errors()
		if ch == nil {
			logger.Error("tunnel.Errors channel is nil!!!!!")
			cancel(constants.ErrChannelIsNil)

			return
		}

		err := <-tunnel.Errors()
		logger.Error("iap tunnel manager returned an error", "error", err)
<<<<<<< Updated upstream
=======
		cancel(err)

		return
>>>>>>> Stashed changes
	}()

	// pass ssh.Dial so we can test with a fake dialer
	if cfg.SshTunnel != nil {
		sshTunnel := iapgo.NewSshTunnel(cfg, ssh.Dial, iapLsnrPort, sshLsnrPort, logger)

		err = sshTunnel.Start(ctx)
		if err != nil {
			logger.Error("failed to start ssh tunnel", "error", err)

			return
		}

		sshLsnrPort = sshTunnel.GetLsnrPort()

		logger.Debug("sshTunnel.Start ran okay")

		defer func() {
			logger.Debug("closing SSH listener")

			_ = sshTunnel.Listener.Close()
		}()
	}

	if cfg.Exec == nil {
		logger.Debug("no Exec command so wait forever.  Enter Control-C to exit.")
		<-ctx.Done()
		if errors.Is(ctx.Err(), context.Canceled) {
			logger.Error("context canceled with error", "error", context.Cause(ctx))
		}
		return
	}

	if cfg.SshTunnel == nil {
		portForRunCmd = iapLsnrPort
	} else {
		portForRunCmd = sshLsnrPort
	}

<<<<<<< Updated upstream
	iapgo.RunCmd(ctx, cfg.Exec, portForRunCmd, logger)
=======
	exec.RunCmd(ctx, cfg.Exec, portForRunCmd, logger)

	if !cfg.TerminateAfterExec {
		logger.Debug("terminate_after_exec is not set so wait forever.  Enter Control-C to exit.")
		<-ctx.Done()
		if errors.Is(ctx.Err(), context.Canceled) {
			logger.Error("context canceled with error", "error", context.Cause(ctx))
		}
	}
>>>>>>> Stashed changes
}
