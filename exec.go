package pocket

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// WaitDelay is the grace period given to child processes to handle
// termination signals before being force-killed.
const WaitDelay = 5 * time.Second

var (
	colorEnvOnce sync.Once
	colorEnvVars []string // extra env vars to force colors
)

// colorForceEnvVars are the environment variables set to force color output.
var colorForceEnvVars = []string{
	"FORCE_COLOR=1",       // Node.js, chalk, many modern tools
	"CLICOLOR_FORCE=1",    // BSD/macOS convention
	"COLORTERM=truecolor", // Indicates color support
}

// computeColorEnv determines which color env vars to use.
// isTTY: whether stdout is a terminal
// noColorSet: whether NO_COLOR env var is set.
func computeColorEnv(isTTY, noColorSet bool) []string {
	// Respect NO_COLOR convention (https://no-color.org/).
	if noColorSet {
		return nil
	}
	// Only force colors if stdout is a terminal.
	if !isTTY {
		return nil
	}
	return colorForceEnvVars
}

// initColorEnv detects if stdout is a TTY and prepares env vars to force colors.
// This is called once on first Command() call.
func initColorEnv() {
	_, noColor := os.LookupEnv("NO_COLOR")
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	colorEnvVars = computeColorEnv(isTTY, noColor)
}

// commandBase creates an exec.Cmd with common setup but no output configuration.
// This is used internally by Command and TaskContext.Command.
func commandBase(ctx context.Context, name string, args ...string) *exec.Cmd {
	colorEnvOnce.Do(initColorEnv)

	binDir := FromBinDir()
	env := PrependPath(os.Environ(), binDir)
	env = append(env, colorEnvVars...)

	// If name is not a path and exists in .pocket/bin/, use the full path.
	// This is needed because exec.Command resolves the binary using os.Getenv("PATH")
	// at creation time, before cmd.Env takes effect.
	if !strings.ContainsAny(name, `/\`) {
		if binPath := filepath.Join(binDir, name); fileExists(binPath) {
			name = binPath
		}
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	setGracefulShutdown(cmd)
	return cmd
}

// fileExists returns true if the path exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// Command creates an exec.Cmd with PATH prepended with .pocket/bin,
// stdout/stderr connected to os.Stdout/os.Stderr, and graceful shutdown configured.
//
// When the context is cancelled, the command receives SIGINT first
// (allowing graceful shutdown), then SIGKILL after WaitDelay.
//
// If stdout is a TTY, color-forcing environment variables are added so that
// tools output ANSI colors even when their output is buffered (for parallel execution).
//
// Note: For commands run from task actions, prefer TaskContext.Command() which
// automatically wires output to the task's output writers for proper parallel buffering.
//
// To redirect output (e.g., for buffering in parallel execution),
// set cmd.Stdout and cmd.Stderr after creating the command.
func Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := commandBase(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
		if oldPath, found := strings.CutPrefix(e, "PATH="); found {
			result = append(result, "PATH="+dir+string(os.PathListSeparator)+oldPath)
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
