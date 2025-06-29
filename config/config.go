package config

type Config struct {
	ProjectID   string   `yaml:"project_id"`
	Zone        string   `yaml:"zone"`
	Instance    string   `yaml:"instance"`
	RemotePort  int      `yaml:"remote_port"`
	LocalPort   int      `yaml:"local_port"`
	RemoteNic   string   `yaml:"remote_nic"`
	Exec        []string `yaml:"exec,omitempty"`
	SshTunnelTo string   `yaml:"ssh_tunnel_to,omitempty"`
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
