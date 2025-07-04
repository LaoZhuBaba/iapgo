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

func dialSshTunnel(
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

type SshTunnel struct {
	config    *Config
	destPort  int
	localPort int
	logger    *slog.Logger
	client    *ssh.Client
	Listener  net.Listener
}

func NewSshTunnel(
	config *Config,
	destPort int,
	localPort int,
	logger *slog.Logger,
) SshTunnel {
	return SshTunnel{
		config:    config,
		destPort:  destPort,
		localPort: localPort,
		logger:    logger,
	}
}

// This method starts the underlying SSH session. It sets the c.client field
// so it requires a pointer receiver.
func (c *SshTunnel) Init(ctx context.Context) error {
	var pkFile string

	if c.config.SshTunnel.PrivateKeyFile == "" {
		pkFile = filepath.Join(os.Getenv("HOME"), ".ssh", "google_compute_engine")
	}

	c.logger.Debug("private key path", "pkFile", pkFile)

	privateKey, err := os.ReadFile(pkFile)
	if err != nil {
		return fmt.Errorf("error reading private key file: %w", err)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("error parsing private key: %w", err)
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
		User: c.config.SshTunnel.AccountName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback:   hostKeyCallback,
		HostKeyAlgorithms: algorithms.HostKeys,
	}

	c.logger.Debug("starting ssh tunnel", "destPort", c.destPort)
	c.client, err = ssh.Dial("tcp", fmt.Sprintf("%s:%d", "localhost", c.destPort), cfg)

	if err != nil {
		return fmt.Errorf("error dialing ssh tunnel: %w", err)
	}

	return nil
}

func (c *SshTunnel) StartTunnel(ctx context.Context) (err error) {
	c.Listener, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", c.config.LocalPort))
	if err != nil {
		return fmt.Errorf("failed to listen (sshLsnr): %w", err)
	}

	c.localPort, err = GetPortFromTcpAddr(c.Listener, c.logger)
	if err != nil {
		return fmt.Errorf("failed to get port from SSH listener: %w", err)
	}

	c.logger.Debug("sshLsnr is listening on TCP port", "port", c.localPort)

	if len(c.config.Exec) == 0 {
		c.logger.Info("no Exec command so enter Control-C to exit")

		for {
			tunnelConn, err := dialSshTunnel(
				c.client,
				c.config.SshTunnel.TunnelTo,
				c.config.RemotePort,
			)
			if err != nil {
				return fmt.Errorf("failed to start ssh tunnel: %w")
			}
			c.logger.Debug(
				"successfully dialled ssh tunnel",
				"TunnelTo", c.config.SshTunnel.TunnelTo,
				"remotePort", c.config.RemotePort,
			)

			localConn, err := c.Listener.Accept()
			if err != nil {
				return fmt.Errorf("failed to accept local connection or listener closed: %w", err)
			}

			go NewHandler(localConn, tunnelConn, c.logger).Handle(ctx)
		}
	} else {
		go func() {
			tunnelConn, err := dialSshTunnel(c.client, c.config.SshTunnel.TunnelTo, c.config.RemotePort)
			if err != nil {
				c.logger.Error("failed to start ssh tunnel", "error", err)
				return
			}
			c.logger.Debug(
				"successfully dialled ssh tunnel",
				"TunnelTo", c.config.SshTunnel.TunnelTo,
				"remotePort", c.config.RemotePort,
			)

			localConn, err := c.Listener.Accept()
			if err != nil {
				c.logger.Error("local listener closed")
				return
			}

			go NewHandler(localConn, tunnelConn, c.logger).Handle(ctx)
		}()
	}
	return nil
}
