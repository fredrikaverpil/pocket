//go:build unix

package engine

import (
	"os/exec"
	"syscall"
)

// SetGracefulShutdown configures the command for graceful shutdown.
// On Unix, this sets up SIGINT as the interrupt signal.
func SetGracefulShutdown(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		return cmd.Process.Signal(syscall.SIGINT)
	}
}
