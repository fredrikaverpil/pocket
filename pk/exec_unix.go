//go:build unix

package pk

import (
	"os/exec"
	"syscall"
)

// setGracefulShutdown configures the command for graceful shutdown.
// On Unix, this sets up SIGINT as the interrupt signal.
func setGracefulShutdown(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		return cmd.Process.Signal(syscall.SIGINT)
	}
}
