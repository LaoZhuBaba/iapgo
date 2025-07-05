package iapgo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/ssh"
)

type SshTunnel struct {
	mu        sync.Mutex
	config    *Config
	destPort  int
	localPort int
	logger    *slog.Logger
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

func (c *SshTunnel) GetLsnrPort() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.localPort
}

func (c *SshTunnel) Start(ctx context.Context) error {
	sshClient, err := c.init()
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server %s via IAP: %w", c.config.Instance, err)
	}

	c.logger.Debug("underlying SSH session started okay")

	c.Listener, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", c.config.LocalPort))
	if err != nil {
		return fmt.Errorf("failed to listen (sshLsnr): %w", err)
	}

	c.localPort, err = GetPortFromTcpAddr(c.Listener, c.logger)
	if err != nil {
		return fmt.Errorf("failed to get port from SSH listener: %w", err)
	}

	c.logger.Debug("sshLsnr is listening on TCP port", "port", c.localPort)

	go c.loop(ctx, sshClient)

	return nil
}

// This method starts the underlying SSH session. It sets the c.client field
// so it requires a pointer receiver.
func (c *SshTunnel) init() (*ssh.Client, error) {
	var pkFile string

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.config.SshTunnel.PrivateKeyFile == "" {
		pkFile = filepath.Join(os.Getenv("HOME"), ".ssh", "google_compute_engine")
	}

	c.logger.Debug("private key path", "pkFile", pkFile)

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
		User: c.config.SshTunnel.AccountName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback:   hostKeyCallback,
		HostKeyAlgorithms: algorithms.HostKeys,
	}

	c.logger.Debug("starting ssh tunnel", "destPort", c.destPort)
	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", "localhost", c.destPort), cfg)

	if err != nil {
		return nil, fmt.Errorf("error dialing ssh tunnel: %w", err)
	}

	return sshClient, nil
}

func (c *SshTunnel) loop(ctx context.Context, client *ssh.Client) {
	for {
		localConn, err := c.Listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				c.logger.Debug("listener closed", "err", err)

				return
			}

			c.logger.Error("error on SSH listener", "err", err)

			return
		}

		c.logger.Debug("SSH tunnel listener accepted a connection", "localPort", c.localPort)

		tunnelConn, err := dialSshTunnel(
			client,
			c.config.SshTunnel.TunnelTo,
			c.config.RemotePort,
		)
		if err != nil {
			c.logger.Error("error dialing ssh tunnel", "err", err)

			return
		}

		c.logger.Debug(
			"successfully dialled ssh tunnel",
			"TunnelTo", c.config.SshTunnel.TunnelTo,
			"remotePort", c.config.RemotePort,
		)

		go NewHandler(localConn, tunnelConn, c.logger).Handle(ctx)
	}
}

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
