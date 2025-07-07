package exec

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

func RunCmd(ctx context.Context, args []string, port int, logger *slog.Logger) {
	// Run the provided command.  To avoid having to enter the local port number into the configuration file twice
	// make it available as an env var.  This will only work if exec runs a shell.  E.g., "bash -c ..."
	err := os.Setenv("IAPGO_LISTEN_PORT", fmt.Sprintf("%d", port))
	if err != nil {
		logger.Error("failed to set IAPGO_LISTEN_PORT environment variable")

		return
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err = cmd.Run()

	if err != nil {
		logger.Error("failed to run command", "error", err)

		return
	}
}
