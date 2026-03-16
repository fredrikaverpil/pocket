package pk

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk/internal/engine"
)

// Exec runs an external command with .pocket/bin prepended to PATH.
// The command runs in the directory from [PathFromContext].
//
// Output handling depends on verbose mode (-v flag):
//   - Verbose: output streams to stdout/stderr in real-time.
//   - Non-verbose: output is captured and shown only on error or when
//     warning-like patterns are detected (see [DefaultNoticePatterns]).
//
// Environment variables can be customized via [ContextWithEnv] and [ContextWithoutEnv].
// On Unix, commands receive SIGINT on context cancellation for graceful shutdown,
// followed by SIGKILL after 5 seconds.
func Exec(ctx context.Context, name string, args ...string) error {
	return engine.Exec(ctx, name, args...)
}

// DefaultNoticePatterns are the substrings used to detect warning-like output
// from commands when not in verbose mode. If any pattern matches (case-insensitive),
// the output is shown to the user even though the command succeeded.
// Override per scope with [WithNoticePatterns].
var DefaultNoticePatterns = engine.DefaultNoticePatterns

// RegisterPATH registers a directory to be added to PATH for all Exec calls.
// Use this for tools that can't be symlinked (e.g., neovim on Windows needs its runtime files).
func RegisterPATH(dir string) {
	engine.RegisterPATH(dir)
}

// Do wraps a Go function as a [Runnable] for use in task composition.
//
//	pk.Do(func(ctx context.Context) error {
//	    return run.Exec(ctx, "golangci-lint", "run", "--fix", "./...")
//	})
func Do(fn func(ctx context.Context) error) Runnable {
	return &doRunnable{fn: fn}
}

// doRunnable wraps a function as a Runnable.
type doRunnable struct {
	fn func(ctx context.Context) error
}

func (d *doRunnable) run(ctx context.Context) error {
	return d.fn(ctx)
}
