package iapgo

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	oslogin "cloud.google.com/go/oslogin/apiv1"
	"cloud.google.com/go/oslogin/apiv1/osloginpb"
)

func GetPosixLogin(ctx context.Context, gcpLogin string) (string, error) {

	osloginClient, err := oslogin.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer osloginClient.Close()

	oslp, err := osloginClient.GetLoginProfile(ctx, &osloginpb.GetLoginProfileRequest{Name: fmt.Sprintf("users/%s", gcpLogin)})
	if err != nil {
		return "", err
	}

	for _, ac := range oslp.PosixAccounts {
		if ac.Primary {
			return ac.Username, nil
		}
	}
	return "", errors.New("no primary posix command could be found")
}

func GetGcpLogin() (string, error) {
	cmd := exec.Command("bash", "-c", "gcloud config get account")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
