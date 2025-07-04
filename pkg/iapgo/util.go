package iapgo

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"strings"

	oslogin "cloud.google.com/go/oslogin/apiv1"
	"cloud.google.com/go/oslogin/apiv1/osloginpb"
)

func getPosixLogin(ctx context.Context, gcpLogin string) (string, error) {
	osloginClient, err := oslogin.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("error getting oslogin client: %w", err)
	}

	defer func() {
		_ = osloginClient.Close()
	}()

	oslp, err := osloginClient.GetLoginProfile(
		ctx,
		&osloginpb.GetLoginProfileRequest{Name: fmt.Sprintf("users/%s", gcpLogin)},
	)
	if err != nil {
		return "", fmt.Errorf("error getting login profile: %w", err)
	}

	for _, ac := range oslp.GetPosixAccounts() {
		if ac.GetPrimary() {
			return ac.GetUsername(), nil
		}
	}

	return "", ErrPrimaryPosixCmdNotFound
}

func getGcpLogin() (string, error) {
	cmd := exec.Command("bash", "-c", "gcloud config get account")
	out, err := cmd.Output()

	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFailedToGetGcpLogin, err)
	}

	return strings.TrimSpace(string(out)), nil
}

func GetPortFromTcpAddr(addr net.Listener, logger *slog.Logger) (int, error) {
	tcpAddr, ok := addr.Addr().(*net.TCPAddr)
	if !ok {
		return 0, ErrNotATcpListener
	}

	return tcpAddr.Port, nil
}
