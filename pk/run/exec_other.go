//go:build !unix

package run

import (
	"os/exec"
)

// setGracefulShutdown configures the command for graceful shutdown.
// On non-Unix platforms, this is a no-op as SIGINT is not available.
func setGracefulShutdown(cmd *exec.Cmd) {
	_ = cmd
}
