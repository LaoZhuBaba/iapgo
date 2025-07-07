package ssh

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"sync"
	"testing"

	"github.com/LaoZhuBaba/iapgo/v2/internal/config"
	"github.com/LaoZhuBaba/iapgo/v2/internal/constants"
	"golang.org/x/crypto/ssh"
)

func test_sshDialerReturnsErr(s1 string, s2 string, config *ssh.ClientConfig) (*ssh.Client, error) {
	return nil, errors.New("random failure")
}

func test_sshDialerReturnsNoErr(s1 string, s2 string, config *ssh.ClientConfig) (*ssh.Client, error) {
	return &ssh.Client{}, nil
}

const privateKeyFilename = "testdata/private.pem"

func TestMain(m *testing.M) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Error generating RSA private key: %v\n", err)
		os.Exit(1)
	}

	// Marshal the private key to PKCS#1 DER format
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)

	// Create a PEM block for the private key
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	// Create the private key file
	privateKeyFile, err := os.Create(privateKeyFilename)
	if err != nil {
		fmt.Printf("Error creating private.pem: %v\n", err)
		os.Exit(1)
	}

	// Encode and write the PEM block to the file
	err = pem.Encode(privateKeyFile, privateKeyBlock)
	if err != nil {
		log.Printf("Error encoding private key to PEM: %v\n", err)
		os.Exit(1)
	}

	privateKeyFile.Close()

	exitCode := m.Run()
	_ = os.Remove(privateKeyFilename)
	os.Exit(exitCode)
}

func TestSshTunnel_Start(t *testing.T) {
	var logLevel slog.LevelVar
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: &logLevel,
	}))
	slog.SetDefault(logger)
	logLevel.Set(slog.LevelInfo)

	type fields struct {
		mu        sync.Mutex
		config    *config.Config
		destPort  int
		localPort int
		logger    *slog.Logger
		Listener  net.Listener
		sshDial   SshDialer
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "ssh_dial_missing_private_key_file",
			fields: fields{
				destPort:  100,
				localPort: 200,
				logger:    logger,
				Listener:  nil,
				sshDial:   test_sshDialerReturnsErr,
				config: &config.Config{
					ProjectID:  "project-id",
					Zone:       "zone",
					Instance:   "instance",
					RemotePort: 100,
					LocalPort:  200,
					RemoteNic:  "remote-nic",
					SshTunnel: &config.SshTunnelCfg{
						TunnelTo:       "tunnel-to",
						AccountName:    "account-name",
						PrivateKeyFile: "does-not-exist",
					},
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: constants.ErrPrivateKeyFileNotFound,
		},
		{
			name: "ssh_dial_invalid_private_key_file",
			fields: fields{
				destPort:  100,
				localPort: 200,
				logger:    logger,
				Listener:  nil,
				sshDial:   test_sshDialerReturnsErr,
				config: &config.Config{
					ProjectID:  "project-id",
					Zone:       "zone",
					Instance:   "instance",
					RemotePort: 100,
					LocalPort:  200,
					RemoteNic:  "remote-nic",
					SshTunnel: &config.SshTunnelCfg{
						TunnelTo:       "tunnel-to",
						AccountName:    "account-name",
						PrivateKeyFile: "testdata/empty_file",
					},
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: constants.ErrInvalidPrivateKeyFile,
		},
		{
			name: "ssh_dial_fails",
			fields: fields{
				destPort:  100,
				localPort: 200,
				logger:    logger,
				Listener:  nil,
				sshDial:   test_sshDialerReturnsErr,
				config: &config.Config{
					ProjectID:  "project-id",
					Zone:       "zone",
					Instance:   "instance",
					RemotePort: 100,
					LocalPort:  200,
					RemoteNic:  "remote-nic",
					SshTunnel: &config.SshTunnelCfg{
						TunnelTo:       "tunnel-to",
						AccountName:    "account-name",
						PrivateKeyFile: privateKeyFilename,
					},
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: constants.ErrSshDialFailed,
		},
		{
			name: "ssh_client_succeeds",
			fields: fields{
				destPort:  100,
				localPort: 2000,
				logger:    logger,
				Listener:  nil,
				sshDial:   test_sshDialerReturnsNoErr,
				config: &config.Config{
					ProjectID:  "project-id",
					Zone:       "zone",
					Instance:   "instance",
					RemotePort: 100,
					LocalPort:  2000,
					RemoteNic:  "remote-nic",
					SshTunnel: &config.SshTunnelCfg{
						TunnelTo:       "tunnel-to",
						AccountName:    "account-name",
						PrivateKeyFile: privateKeyFilename,
					},
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &SshTunnel{
				mu:        tt.fields.mu,
				config:    tt.fields.config,
				destPort:  tt.fields.destPort,
				localPort: tt.fields.localPort,
				logger:    tt.fields.logger,
				Listener:  tt.fields.Listener,
				sshDial:   tt.fields.sshDial,
			}
			fmt.Printf("config: %+v\n", c.config)
			fmt.Printf("SSH config: %+v\n", *c.config.SshTunnel)
			if err := c.Start(tt.args.ctx); !errors.Is(err, tt.wantErr) {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
