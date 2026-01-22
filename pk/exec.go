package pk

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// WaitDelay is the time to wait after sending SIGINT before sending SIGKILL.
const WaitDelay = 5 * time.Second

// Do creates a Runnable that executes a dynamic function.
// Use this when command arguments need to be computed at runtime.
//
//	pk.Do(func(ctx context.Context) error {
//	    return pk.Exec(ctx, "golangci-lint", "run", "--fix", "./...")
//	})
func Do(fn func(ctx context.Context) error) Runnable {
	return &doRunnable{fn: fn}
}

var (
	colorEnvOnce sync.Once
	colorEnvVars []string
)

// colorForceEnvVars are the environment variables set to force color output.
var colorForceEnvVars = []string{
	"FORCE_COLOR=1",       // Node.js, chalk, many modern tools
	"CLICOLOR_FORCE=1",    // BSD/macOS convention
	"COLORTERM=truecolor", // Indicates color support
}

// initColorEnv detects if stdout is a TTY and prepares env vars to force colors.
func initColorEnv() {
	_, noColor := os.LookupEnv("NO_COLOR")
	if noColor {
		return
	}

	if isTerminal(os.Stdout) {
		colorEnvVars = colorForceEnvVars
	}
}

// isTerminal returns true if the given file is a terminal.
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// Exec executes a command with .pocket/bin prepended to PATH.
// This ensures tools installed via InstallGo() are found first.
// The command runs in the directory specified by PathFromContext(ctx).
//
// If verbose mode is enabled, command output is streamed to context output.
// Otherwise, output is captured and only shown on error.
//
// Commands are terminated gracefully: SIGINT first, then SIGKILL after WaitDelay.
func Exec(ctx context.Context, name string, args ...string) error {
	colorEnvOnce.Do(initColorEnv)

	path := PathFromContext(ctx)
	targetDir := FromGitRoot(path)

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = targetDir
	cmd.Env = prependBinToPath(os.Environ())
	cmd.Env = append(cmd.Env, colorEnvVars...)
	cmd.WaitDelay = WaitDelay
	setGracefulShutdown(cmd)

	out := OutputFromContext(ctx)

	if Verbose(ctx) {
		cmd.Stdout = out.Stdout
		cmd.Stderr = out.Stderr
		return cmd.Run()
	}

	// Capture output and only show on error.
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	if err != nil {
		// Include output in error for debugging.
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, buf.String())
	}
	return nil
}

// prependBinToPath adds .pocket/bin to the front of PATH.
func prependBinToPath(environ []string) []string {
	binDir := FromBinDir()

	result := make([]string, 0, len(environ))
	for _, env := range environ {
		if strings.HasPrefix(env, "PATH=") {
			existingPath := strings.TrimPrefix(env, "PATH=")
			result = append(result, fmt.Sprintf("PATH=%s%c%s", binDir, filepath.ListSeparator, existingPath))
		} else {
			result = append(result, env)
		}
	}
	return result
}

// doRunnable wraps a function as a Runnable.
type doRunnable struct {
	fn func(ctx context.Context) error
}

func (d *doRunnable) run(ctx context.Context) error {
	return d.fn(ctx)
}
