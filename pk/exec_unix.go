//go:build unix

package pk

import (
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/term"
)

// setGracefulShutdown configures the command for graceful shutdown.
// On Unix, this sets up SIGINT as the interrupt signal.
func setGracefulShutdown(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		return cmd.Process.Signal(syscall.SIGINT)
	}
}

// isTerminal returns true if the given file is a terminal.
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
