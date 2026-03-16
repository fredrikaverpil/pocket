//go:build !unix

package engine

import (
	"os/exec"
)

// SetGracefulShutdown configures the command for graceful shutdown.
// On non-Unix platforms, this is a no-op as SIGINT is not available.
func SetGracefulShutdown(cmd *exec.Cmd) {
	_ = cmd
}
