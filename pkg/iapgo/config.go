package iapgo

import (
	"context"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ProjectID  string     `yaml:"project_id"`
	Zone       string     `yaml:"zone"`
	Instance   string     `yaml:"instance"`
	RemotePort int        `yaml:"remote_port"`
	LocalPort  int        `yaml:"local_port"`
	RemoteNic  string     `yaml:"remote_nic"`
	Exec       []string   `yaml:"exec,omitempty"`
	SshTunnel  *SshTunnel `yaml:"ssh_tunnel,omitempty"`
}

type SshTunnel struct {
	TunnelTo       string `yaml:"tunnel_to"`
	AccountName    string `yaml:"account_name,omitempty"`
	PrivateKeyFile string `yaml:"private_key_file,omitempty"`
}

const ExampleConfig = `
	default:
	  project_id: my-gcp-project
	  zone: us-central1-a
	  instance: my-jumpbox
	  local_port: 1234
	  remote_port: 2345
	  remote_nic: nic0
	  exec:
		- bash
		- "-c"
		- my-command
	  ssh_tunnel:
		tunnel_to: 1.2.3.4
		private_key_file: /home/fred/.ssh/rsa_key
	example:
	  project_id: my-gcp-project
	  zone: us-central1-a
	  instance: my-jumpbox
	  local_port: 1111
	  remote_port: 1111
	  remote_nic: nic0
	  exec:
		- bash
		- "-c"
		- my-command
`

func GetConfig(ctx context.Context, yamlFileName string, cfgSection string, logger *slog.Logger) (*Config, error) {
	var cfgMap map[string]Config

	yamlFile, err := os.ReadFile(yamlFileName)
	if err != nil {
		logger.Error("failed to read YAML file with error", "error", err)
		return nil, err
	}

	err = yaml.Unmarshal(yamlFile, &cfgMap)
	if err != nil {
		logger.Error("Error unmarshalling YAML", "error", err)
		return nil, err
	}
	cfg, ok := cfgMap[cfgSection]
	if !ok {
		logger.Error("config section does not exist", "*configSectionPtr", cfgSection)
		return nil, err
	}

	if cfg.SshTunnel != nil {
		if cfg.SshTunnel.TunnelTo == "" {
			logger.Error("if ssh_tunnel is configured then ssh_tunnel_to must have a value")
			return nil, err
		}
	}

	if cfg.SshTunnel != nil {
		if cfg.SshTunnel.TunnelTo == "" {
			logger.Error("if ssh_tunnel is configured then ssh_tunnel_to must have a value")
			return nil, err
		}
	}

	if cfg.SshTunnel != nil && cfg.SshTunnel.AccountName == "" {
		logger.Debug("no posix account name found in config so attempting to resolve from OS Login")
		login, err := GetGcpLogin()
		if err != nil {
			logger.Error("failed to get gcp login", "error", err)
			logger.Error("this may be because the 'gcloud' command is not in your path or you are not logged into GCP")
			return nil, err
		}

		cfg.SshTunnel.AccountName, err = GetPosixLogin(ctx, login)
		if err != nil {
			logger.Error("failed to get posix login", "error", err)
			return nil, err
		}
		logger.Debug("successfully resolved from OS Login", "AccountName", cfg.SshTunnel.AccountName)
	}

	return &cfg, nil
}
