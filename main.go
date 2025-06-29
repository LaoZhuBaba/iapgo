package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"

	config "github.com/LaoZhuBaba/iapgo/config"
	tunnel "github.com/davidspek/go-iap-tunnel/pkg"
	"golang.org/x/crypto/ssh"
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

	// Start the real work here
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// portCh is needed when using SSH tunnelling because in that case IAP listens on an ephemeral local port
	portCh := make(chan int)
	// errChh is for reporting errors, must have a buffer of 1 to avoid a deadlock
	errCh := make(chan error)
	var port int

	go func() {
		logger.Info("starting IAP tunnel")
		startIapTunnel(ctx, cfg, logger, portCh, errCh)
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

	if cfg.SshTunnelTo != "" {
		startSshTunnel(ctx, cfg, port, logger, errCh)
	}

	if len(cfg.Exec) > 0 {
		go func() {
			logger.Debug("running command")
			runCmd(ctx, cfg.Exec, logger, errCh)
			logger.Debug("command ended so cancelling context")
			cancel()
		}()

	}

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

func runCmd(ctx context.Context, args []string, logger *slog.Logger, errCh chan<- error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		logger.Error("failed to run command", "error", err)
		errCh <- err
	}
}

func startSshTunnel(ctx context.Context, iapCfg config.Config, destPort int, logger *slog.Logger, errCh chan error) {
	logger.Info("top of startSshTunnel", "destPort", destPort)
	key, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".ssh", "google_compute_engine"))
	if err != nil {
		logger.Error("unable to read private key", "error", err)
		errCh <- err
		return
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		logger.Error("unable to parse private key", "error", err)
		errCh <- err
		return
	}

	hostKeyCallback := ssh.InsecureIgnoreHostKey()

	algorithms := ssh.SupportedAlgorithms()
	cfg := &ssh.ClientConfig{
		Config: ssh.Config{
			KeyExchanges: algorithms.KeyExchanges,
			Ciphers:      algorithms.Ciphers,
			MACs:         algorithms.MACs,
		},
		User: "adm_david_liebert_qoria_com",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
			//ssh.Password("ssh@test"),
		},
		HostKeyCallback:   hostKeyCallback,
		HostKeyAlgorithms: algorithms.HostKeys,
	}

	logger.Debug("starting ssh tunnel", "destPort", destPort)
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", "localhost", destPort), cfg)
	if err != nil {
		logger.Error("Failed to dial SSH tunnel", "error", err)
	}
	logger.Debug("after starting ssh tunnel")

	tunnelConn, err := client.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(iapCfg.SshTunnelTo), Port: iapCfg.RemotePort})
	if err != nil {
		logger.Error("Failed to dial SSH tunnel", "error", err)
	}
	logger.Debug("after DialTCP")

	l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", iapCfg.LocalPort))
	if err != nil {
		logger.Error("Failed to listen", "error", err)
	}
	logger.Debug("after Listen", "addr", l.Addr().String())
	go func() {
		conn, err := l.Accept()
		if err != nil {
			logger.Error("Failed to accept SSH tunnel", "error", err)
		}

		logger.Debug("after Accept")
		go func() {
			_, err := io.Copy(tunnelConn, conn)
			if err != nil {
				errCh <- err
			}
		}()

		go func() {
			_, err := io.Copy(conn, tunnelConn)
			if err != nil {
				errCh <- err
			}
		}()
	}()
	logger.Debug("after starting Copies")
}

func startIapTunnel(ctx context.Context, conf config.Config, logger *slog.Logger, portCh chan<- int, errCh chan<- error) {
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

		logger.Debug("starting IAP server", "port", listener.Addr().(*net.TCPAddr).Port)
		err = manager.Serve(ctx, listener)
		if err != nil {
			logger.Error("failed to start tunnel", "error", err)
			return
		}
		logger.Debug("after starting IAP server", "port", listener.Addr().(*net.TCPAddr).Port)

	}()
	//<-ctx.Done()
}
