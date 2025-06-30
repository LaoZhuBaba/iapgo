package iapgo

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

func RunCmd(ctx context.Context, args []string, port int, logger *slog.Logger, errCh chan<- error) {
	// Run the provided command.  To avoid having to enter the local port number into the configuration file twice
	// make it available as an env var.  This will only work if exec runs a shell.  E.g., "bash -c ..."
	os.Setenv("IAPGO_LISTEN_PORT", fmt.Sprintf("%d", port))
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		logger.Error("failed to run command", "error", err)
		errCh <- err
	}
}
