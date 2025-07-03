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

func StartSshTunnel(
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
