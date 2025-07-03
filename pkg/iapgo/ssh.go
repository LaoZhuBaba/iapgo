package iapgo

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

func DialSshTunnel(
	ctx context.Context,
	client *ssh.Client,
	destAddr string,
	destPort int,
) (net.Conn, error) {
	conn, err := client.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(destAddr), Port: destPort})
	if err != nil {
		return conn, fmt.Errorf("error starting ssh tunnel: %w", err)
	}
	return conn, nil
}

func CreateSshClient(
	ctx context.Context,
	iapCfg Config,
	posixAccount string,
	destPort int,
	logger *slog.Logger,
) (*ssh.Client, error) {
	var pkFile string

	if iapCfg.SshTunnel.PrivateKeyFile == "" {
		pkFile = filepath.Join(os.Getenv("HOME"), ".ssh", "google_compute_engine")
	} else {
		pkFile = iapCfg.SshTunnel.PrivateKeyFile
	}

	logger.Debug("private key path", "pkFile", pkFile)

	privateKey, err := os.ReadFile(pkFile)
	if err != nil {
		return nil, fmt.Errorf("error reading private key file: %w", err)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("error parsing private key: %w", err)
	}

	// This disables normal checking to ensure that the host you are connecting matches the host recorded
	// in known_hosts.  But in our case we are connecting via an IAP tunnel so the host is already trusted.
	hostKeyCallback := ssh.InsecureIgnoreHostKey()

	algorithms := ssh.SupportedAlgorithms()
	cfg := &ssh.ClientConfig{
		Config: ssh.Config{
			KeyExchanges: algorithms.KeyExchanges,
			Ciphers:      algorithms.Ciphers,
			MACs:         algorithms.MACs,
		},
		User: posixAccount,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback:   hostKeyCallback,
		HostKeyAlgorithms: algorithms.HostKeys,
	}

	logger.Debug("starting ssh tunnel", "destPort", destPort)
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", "localhost", destPort), cfg)

	if err != nil {
		return nil, fmt.Errorf("error dialing ssh tunnel: %w", err)
	}

	return client, nil
}

func StartSshTunnel(
	ctx context.Context,
	cfg *Config,
	iapLsnrPort int,
	sshLsnrPort int,
	logger *slog.Logger,
) (net.Listener, error) {
	sshClient, err := CreateSshClient(
		ctx,
		*cfg,
		cfg.SshTunnel.AccountName,
		iapLsnrPort,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start ssh client: %w", err)
	}

	logger.Debug("CreateSshClient ran okay")

	sshLsnr, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", cfg.LocalPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen (sshLsnr): %w", err)
	}

	sshLsnrPort, err = GetPortFromTcpAddr(sshLsnr, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get port from SSH listener: %w", err)
	}

	logger.Debug("sshLsnrAddrr is listening on TCP port", "port", sshLsnrPort)

	if len(cfg.Exec) == 0 {
		logger.Info("no Exec command so enter Control-C to exit")

		for {
			tunnelConn, err := DialSshTunnel(
				ctx,
				sshClient,
				cfg.SshTunnel.TunnelTo,
				cfg.RemotePort,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to start ssh tunnel: %w")
			}

			localConn, err := sshLsnr.Accept()
			if err != nil {
				return nil, fmt.Errorf("failed to accept local connection or listener closed: %w", err)
			}

			go NewHandler(localConn, tunnelConn, logger).Handle(ctx)
		}
	} else {
		go func() {
			tunnelConn, err := DialSshTunnel(ctx, sshClient, cfg.SshTunnel.TunnelTo, cfg.RemotePort)
			if err != nil {
				logger.Error("failed to start ssh tunnel", "error", err)
				return
			}

			localConn, err := sshLsnr.Accept()
			if err != nil {
				logger.Error("local listener closed")
				return
			}

			go NewHandler(localConn, tunnelConn, logger).Handle(ctx)
		}()
	}
	return sshLsnr, nil
}
