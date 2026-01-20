package pk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Do creates a Runnable that executes a dynamic function.
// Use this when command arguments need to be computed at runtime.
//
//	pk.Do(func(ctx context.Context) error {
//	    return pk.Exec(ctx, "golangci-lint", "run", "--fix", "./...")
//	})
func Do(fn func(ctx context.Context) error) Runnable {
	return &doRunnable{fn: fn}
}

// Exec executes a command with .pocket/bin prepended to PATH.
// This ensures tools installed via InstallGo() are found first.
// The command runs in the directory specified by PathFromContext(ctx).
//
// If verbose mode is enabled, command output is streamed to stdout/stderr.
// Otherwise, output is captured and only shown on error.
func Exec(ctx context.Context, name string, args ...string) error {
	path := PathFromContext(ctx)
	targetDir := FromGitRoot(path)

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = targetDir
	cmd.Env = prependBinToPath(os.Environ())

	if Verbose(ctx) {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Include output in error for debugging
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, output)
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
