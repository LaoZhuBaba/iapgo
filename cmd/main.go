package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/LaoZhuBaba/iapgo/pkg/iapgo"

	yaml "gopkg.in/yaml.v3"
)

const (
	defaultConfigFileName = "iapgo.yaml"
	defaultConfigSection  = "default"
)

func abnormalExit() {
	flag.Usage()
	os.Exit(1)
}

func main() {
	// Trap if we are killed by Control-C or similar
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	helpPtr := flag.Bool("h", false, "print a usage message")
	configSectionPtr := flag.String("c", defaultConfigSection, "select a non-default configuration file section")
	configFilePtr := flag.String("f", defaultConfigFileName, "select a non-default configuration file")
	verbosePtr := flag.Bool("v", false, "print debugging messages")

	flag.Parse()

	if *helpPtr {
		flag.Usage()
		fmt.Println("\nExample configuration file...")
		fmt.Print(iapgo.ExampleConfig)
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

	yamlFile, err := os.ReadFile(*configFilePtr)
	if err != nil {
		logger.Error("failed to read YAML file with error", "error", err)
		abnormalExit()
	}

	var cfgMap map[string]iapgo.Config

	err = yaml.Unmarshal(yamlFile, &cfgMap)
	if err != nil {
		logger.Error("Error unmarshalling YAML", "error", err)
		abnormalExit()
	}
	cfg, ok := cfgMap[*configSectionPtr]
	if !ok {
		logger.Error("config section does not exist", "*configSectionPtr", *configSectionPtr)
		abnormalExit()
	}

	if cfg.SshTunnel != nil {
		if cfg.SshTunnel.TunnelTo == "" {
			logger.Error("if ssh_tunnel is configured then ssh_tunnel_to must have a value")
			abnormalExit()
		}
	}
	logger.Debug("config", "cfgMap[*configSectionPtr]", cfgMap[*configSectionPtr])

	// Start the real work here
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var posixAccount string

	if cfg.SshTunnel.AccountName != "" {
		posixAccount = cfg.SshTunnel.AccountName
	} else {
		login, err := iapgo.GetGcpLogin()
		if err != nil {
			logger.Error("failed to get gcp login", "error", err)
			logger.Error("this may be because the 'gcloud' command is not in your path or you are not logged into GCP")
		}

		posixAccount, err = iapgo.GetPosixLogin(ctx, login)
		if err != nil {
			logger.Error("failed to get posix login", "error", err)
		}
	}
	fmt.Println("posix login:", posixAccount)

	// portCh is needed when using SSH tunnelling because in that case IAP listens on an ephemeral local port
	portCh := make(chan int)
	// errChh is for reporting errors, must have a buffer of 1 to avoid a deadlock
	errCh := make(chan error)
	var port int

	go func() {
		logger.Info("starting IAP tunnel")
		iapgo.StartIapTunnel(ctx, cfg, logger, portCh, errCh)
		logger.Info("IAP tunnel started")
	}()

	// Any error that occurs in startIapTunnel is fatal.  The expected path to receive a port
	select {
	case port = <-portCh:
		logger.Info("tunnel listening on port", "port", port)
	case err := <-errCh:
		logger.Error("tunnel exited with error", "error", err)
		return
	case <-ctx.Done():
		logger.Info("context canceled while waiting for IAP tunnel startup", "error", ctx.Err())
		return
	}

	// SSH tunnelling is optional
	if cfg.SshTunnel != nil {
		err = iapgo.StartSshTunnel(ctx, cfg, posixAccount, port, logger, errCh)
		if err != nil {
			logger.Error("failed to start ssh tunnel", "error", err)
			return
		}
	}

	// The Exec option is optional
	if len(cfg.Exec) > 0 {
		go func() {
			logger.Debug("running command")
			iapgo.RunCmd(ctx, cfg.Exec, cfg.LocalPort, logger, errCh)
			logger.Debug("command ended so cancelling context")
			cancel()
		}()

	} else {
		logger.Info("tunnel start.  Ctrl-C to exit")
	}

	// If the Exec option is enabled then the context will be cancelled after runCmd() completes.  If Exec
	// is not enabled then the tunnel will stay open until an error occurs or context cancellation.
	select {
	case err, ok := <-errCh:
		if !ok {
			logger.Error("tunnel exited without error", "error", err)
			return
		}
		logger.Error("tunnel exited with error", "error", err)
		return
	case <-ctx.Done():
		logger.Info("exiting because context canceled")
		return
	}
}
