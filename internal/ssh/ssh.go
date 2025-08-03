package ssh

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/LaoZhuBaba/iapgo/v2/internal/config"
	"github.com/LaoZhuBaba/iapgo/v2/internal/constants"
	"github.com/LaoZhuBaba/iapgo/v2/internal/util"
	"golang.org/x/crypto/ssh"
)

type SshDialer func(network string, addr string, config *ssh.ClientConfig) (*ssh.Client, error)

type SshTunnel struct {
	mu        sync.Mutex
	config    *config.Config
	destPort  int
	localPort int
	logger    *slog.Logger
	Listener  net.Listener
	sshDial   SshDialer
}

func NewSshTunnel(
	config *config.Config,
	sshDial SshDialer,
	destPort int,
	localPort int,
	logger *slog.Logger,
) SshTunnel {
	return SshTunnel{
		config:    config,
		destPort:  destPort,
		localPort: localPort,
		logger:    logger,
		sshDial:   sshDial,
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
		return fmt.Errorf("%w: %w", constants.ErrSshDialFailed, err)
	}

	c.logger.Debug("underlying SSH session started okay")

	c.Listener, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", c.config.LocalPort))
	if err != nil {
		return fmt.Errorf("%w (sshLsnr): %w", constants.ErrFailedToListen, err)
	}

	c.localPort, err = util.GetPortFromTcpAddr(c.Listener, c.logger)
	if err != nil {
		return fmt.Errorf("%w: %w", constants.ErrFailedToGetPort, err)
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
	} else {
		pkFile = c.config.SshTunnel.PrivateKeyFile
	}

	c.logger.Debug("private key path", "pkFile", pkFile)

	privateKey, err := os.ReadFile(pkFile)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", constants.ErrPrivateKeyFileNotFound, err)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", constants.ErrInvalidPrivateKeyFile, err)
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

	// Any error from sshDial() is hendled in the calling function.
	return c.sshDial("tcp", fmt.Sprintf("%s:%d", "localhost", c.destPort), cfg)
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

		tunnelConn, err := c.dialSshTunnel(client)
		if err != nil {
			c.logger.Error("error dialing ssh tunnel", "err", err)

			return
		}

		c.logger.Debug(
			"successfully dialled ssh tunnel",
			"TunnelTo", c.config.SshTunnel.TunnelTo,
			"remotePort", c.config.RemotePort,
		)

		go func() {
			err1, err2 := NewHandler(localConn, tunnelConn, c.logger).Handle()
			c.logger.Debug("handler exited", "local conn error", err1, "tunnel conn error", err2)
		}()
	}
}

func (c *SshTunnel) dialSshTunnel(
	client *ssh.Client,
) (net.Conn, error) {
	conn, err := client.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(c.config.SshTunnel.TunnelTo), Port: c.config.RemotePort})
	if err != nil {
		return conn, fmt.Errorf("error starting ssh tunnel: %w", err)
	}

	return conn, nil
}
