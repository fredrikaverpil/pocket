//go:build !unix

package pk

import (
	"os"
	"os/exec"
)

// setGracefulShutdown configures the command for graceful shutdown.
// On non-Unix platforms, this is a no-op as SIGINT is not available.
// The command will be terminated using the default mechanism.
func setGracefulShutdown(cmd *exec.Cmd) {
	// No-op on non-Unix platforms.
	// cmd.Cancel defaults to os.Process.Kill.
	_ = cmd // silence unused parameter warning
}

// isTerminal returns true if the given file is a terminal.
func isTerminal(f *os.File) bool {
	// Default to false on non-Unix platforms for now.
	return false
}
