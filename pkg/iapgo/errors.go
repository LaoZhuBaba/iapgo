package iapgo

import "errors"

var (
	ErrSshDialFailed          = errors.New("error dialing ssh tunnel")
	ErrFailedToReadYaml       = errors.New("failed to read yaml file")
	ErrFailedToUnmarshalYaml  = errors.New("failed to unmarshal yaml file")
	ErrConfigSectionNotFound  = errors.New("config section not found")
	ErrNotATcpListener        = errors.New("not a TCP listener")
	ErrPrimaryPosixAcNotFound = errors.New("primary Posix account not found")
	ErrFailedToGetGcpLogin    = errors.New("failed to get GCP login")
	ErrTunnelReadyTimeout     = errors.New("timed out waiting for the tunnel to be ready")
	ErrTunnelReturnedError    = errors.New("tunnel returned an error")
	ErrSshTunnelToNoValue     = errors.New("ssh_tunnel_to must have a value")
	ErrChannelIsNil           = errors.New("channel is nil")
	ErrFailedToListen         = errors.New("failed to listen")
	ErrFailedToGetPort        = errors.New("failed to get port")
	ErrPrivateKeyFileNotFound = errors.New("private key file not found")
	ErrInvalidPrivateKeyFile  = errors.New("invalid private key file")
)
