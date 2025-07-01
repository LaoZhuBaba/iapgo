package iapgo

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

func StartSshTunnel(ctx context.Context, iapCfg Config, posixAccount string, destPort int, logger *slog.Logger, errCh chan error) error {
	key, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".ssh", "google_compute_engine"))
	if err != nil {
		return err
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return err
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
		return err
	}

	tunnelConn, err := client.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(iapCfg.SshTunnel.TunnelTo), Port: iapCfg.RemotePort})
	if err != nil {
		return err
	}

	l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", iapCfg.LocalPort))
	if err != nil {
		return err
	}
	logger.Debug("accepting connections")
	go func() {
		conn, err := l.Accept()
		if err != nil {
			logger.Error("Failed to accept SSH tunnel", "error", err)
			errCh <- err
		}

		logger.Debug("copying local data into tunnel")
		go func() {
			_, err := io.Copy(tunnelConn, conn)
			if err != nil {
				errCh <- err
			}
		}()

		logger.Debug("copying data from tunnel to local")
		go func() {
			_, err := io.Copy(conn, tunnelConn)
			if err != nil {
				errCh <- err
			}
		}()
	}()
	return nil
}
