package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"

	config "github.com/LaoZhuBaba/iapgo/config"
	tunnel "github.com/davidspek/go-iap-tunnel/pkg"
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
		fmt.Print(config.ExampleConfig)
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

	var cfgMap map[string]config.Config

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

	logger.Debug("config", "cfgMap[*configSectionPtr]", cfgMap[*configSectionPtr])

	ctx, cancel := context.WithCancel(ctx)

	go func() {
		logger.Info("starting tunnel")
		defer logger.Info("ending tunnel")
		startTunnel(ctx, cfg, logger)
	}()

	if len(cfg.Exec) > 0 {
		go func() {
			logger.Debug("running command")
			defer logger.Debug("ending command")

			runCmd(ctx, cfg.Exec, logger)
			cancel()
		}()

	}
	<-ctx.Done()
}

func runCmd(ctx context.Context, args []string, logger *slog.Logger) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		logger.Error("failed to run command", "error", err)
	}
}

func startTunnel(ctx context.Context, conf config.Config, logger *slog.Logger) {
	go func() {
		target := tunnel.TunnelTarget{
			Project:   conf.ProjectID,
			Zone:      conf.Zone,
			Instance:  conf.Instance,
			Port:      conf.RemotePort,
			Interface: conf.RemoteNic,
		}

		listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", conf.LocalPort))
		if err != nil {
			logger.Error("failed to listen", "error", err)
			abnormalExit()
		}

		manager := tunnel.NewTunnelManager(target, nil)

		err = manager.Serve(context.Background(), listener)
		if err != nil {
			logger.Error("failed to start tunnel", "error", err)
			abnormalExit()
		}
	}()
	<-ctx.Done()
}
