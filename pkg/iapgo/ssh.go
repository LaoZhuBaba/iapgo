package iapgo

import (
	"context"
	"fmt"
	"golang.org/x/crypto/ssh"
	"log/slog"
	"net"
	"os"
	"path/filepath"
)

func StartSshTunnel(ctx context.Context, client *ssh.Client, destAddr string, destPort int) (net.Conn, error) {
	return client.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(destAddr), Port: destPort})
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
		return nil, err
	}

	// Create the Signer for this private private key.
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	return client, err
}
