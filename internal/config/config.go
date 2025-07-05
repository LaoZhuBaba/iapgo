package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/LaoZhuBaba/iapgo/v2/internal/constants"
	"github.com/LaoZhuBaba/iapgo/v2/internal/utils"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ProjectID  string        `yaml:"project_id"`
	Zone       string        `yaml:"zone"`
	Instance   string        `yaml:"instance"`
	RemotePort int           `yaml:"remote_port"`
	LocalPort  int           `yaml:"local_port"`
	RemoteNic  string        `yaml:"remote_nic"`
	Exec       []string      `yaml:"exec,omitempty"`
	SshTunnel  *SshTunnelCfg `yaml:"ssh_tunnel,omitempty"`
}

type SshTunnelCfg struct {
	TunnelTo       string `yaml:"tunnel_to"`
	AccountName    string `yaml:"account_name,omitempty"`
	PrivateKeyFile string `yaml:"private_key_file,omitempty"`
}

// This is printed out as part of the "usage" output.
const ExampleConfig = `
# default will be used if no config section is specified
default:
  project_id: my-gcp-project
  zone: us-central1-a
  instance: my-jumpbox
  # If local_port is not set then an ephemeral port will be allocated and made available as $IAP_LISTEN_PORT
  # local_port: 1234
  remote_port: 80
  remote_nic: nic0
  exec:
    - bash
    - "-c"
    - curl http://localhost:$IAP_LISTEN_PORT
example:
  project_id: my-gcp-project
  zone: us-central1-a
  instance: my-jumpbox
  remote_port: 80 # When ssh_tunnel is used then this is the port on the tunnel_to hosts
  remote_nic: nic0
  ssh_tunnel:
    tunnel_to: 1.2.3.4 # This is a host that is reachable from my-jumpbox
    # If account_name is not set then an attempt will be made to get value from os-login
    # account_name: my_ssh_login
    # By default ~/.ssh/google_compute_engine will be used.
    # private_key_file: /home/fred/.ssh/google_compute_engine
  exec:
    - bash
    - "-c"
    # curl will reach ssh_tunnel.tunnel_to host on remote_port
    - curl http://localhost:$IAP_LISTEN_PORT
`

func GetConfig(
	ctx context.Context,
	yamlFileName string,
	cfgSection string,
	logger *slog.Logger,
) (*Config, error) {
	var cfgMap map[string]Config

	yamlFile, err := os.ReadFile(yamlFileName)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", constants.ErrFailedToReadYaml, err)
	}

	err = yaml.Unmarshal(yamlFile, &cfgMap)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", constants.ErrFailedToUnmarshalYaml, err)
	}

	cfg, ok := cfgMap[cfgSection]

	if !ok {
		return nil, fmt.Errorf("%w: %s", constants.ErrConfigSectionNotFound, cfgSection)
	}

	if cfg.SshTunnel != nil {
		if cfg.SshTunnel.TunnelTo == "" {
			return nil, constants.ErrSshTunnelToNoValue
		}
	}

	if cfg.SshTunnel != nil && cfg.SshTunnel.AccountName == "" {
		logger.Debug("no posix account name found in config so attempting to resolve from OS Login")

		login, err := utils.GetGcpLogin()
		if err != nil {
			logger.Error("failed to get gcp login", "error", err)
			logger.Error(
				"this may be because the 'gcloud' command is not in your path or you are not logged into GCP",
			)

			return nil, err
		}

		cfg.SshTunnel.AccountName, err = utils.GetPosixLogin(ctx, login)
		if err != nil {
			logger.Error("failed to get posix login", "error", err)

			return nil, err
		}

		logger.Debug(
			"successfully resolved from OS Login",
			"AccountName",
			cfg.SshTunnel.AccountName,
		)
	}

	return &cfg, nil
}
