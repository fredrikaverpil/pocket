package pocket

import (
	"context"
	"os"
	"os/exec"
	"time"
)

// WaitDelay is the grace period given to child processes to handle
// termination signals before being force-killed.
const WaitDelay = 5 * time.Second

// Command creates an exec.Cmd with PATH prepended with .pocket/bin,
// stdout/stderr connected, and graceful shutdown configured.
//
// When the context is cancelled, the command receives SIGINT first
// (allowing graceful shutdown), then SIGKILL after WaitDelay.
func Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = PrependPath(os.Environ(), FromBinDir())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	setGracefulShutdown(cmd)
	return cmd
}

// setGracefulShutdown configures a command for graceful shutdown.
// When the context is cancelled, the process receives SIGINT first,
// then SIGKILL after WaitDelay if still running.
func setGracefulShutdown(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		return cmd.Process.Signal(os.Interrupt)
	}
	cmd.WaitDelay = WaitDelay
}

// PrependPath prepends a directory to the PATH in the given environment.
func PrependPath(env []string, dir string) []string {
	result := make([]string, 0, len(env)+1)
	pathSet := false
	for _, e := range env {
		if len(e) > 5 && e[:5] == "PATH=" {
			result = append(result, "PATH="+dir+string(os.PathListSeparator)+e[5:])
			pathSet = true
		} else {
			result = append(result, e)
		}
	}
	if !pathSet {
		result = append(result, "PATH="+dir)
	}
	return result
}
