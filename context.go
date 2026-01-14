package pocket

import (
	"context"
	"fmt"
	"os/exec"
)

// execContext holds runtime state for function execution.
// Stored in context.Context and accessed via helper functions.
type execContext struct {
	out      *Output          // where to write output
	path     string           // current path for this invocation
	cwd      string           // where CLI was invoked (relative to git root)
	verbose  bool             // verbose mode enabled
	opts     map[string]any   // func name -> options struct
	executed map[uintptr]bool // tracks executed runnables for deduplication
}

// contextKey is the type for context keys to avoid collisions.
type contextKey int

const (
	execContextKey contextKey = iota
)

// newExecContext creates a new execution context.
func newExecContext(out *Output, cwd string, verbose bool) *execContext {
	return &execContext{
		out:      out,
		cwd:      cwd,
		verbose:  verbose,
		opts:     make(map[string]any),
		executed: make(map[uintptr]bool),
	}
}

// withExecContext returns a new context with the execContext attached.
func withExecContext(ctx context.Context, ec *execContext) context.Context {
	return context.WithValue(ctx, execContextKey, ec)
}

// getExecContext retrieves the execContext from the context.
// Panics if not present (programming error).
func getExecContext(ctx context.Context) *execContext {
	ec, ok := ctx.Value(execContextKey).(*execContext)
	if !ok {
		panic("pocket: execContext not found in context - are you calling from outside pocket.Func?")
	}
	return ec
}

// withPath returns a context with the path set for the current invocation.
func withPath(ctx context.Context, path string) context.Context {
	ec := getExecContext(ctx)
	// Create a shallow copy with the new path
	newEC := *ec
	newEC.path = path
	return withExecContext(ctx, &newEC)
}

// withOptions stores options for a function in the context.
func withOptions(ctx context.Context, name string, opts any) context.Context {
	ec := getExecContext(ctx)
	ec.opts[name] = opts
	return ctx
}

// Exec runs an external command with output directed to the current context.
// The command runs in the current path directory.
//
// Example:
//
//	func goFormat(ctx context.Context) error {
//	    return pocket.Exec(ctx, "go", "fmt", "./...")
//	}
func Exec(ctx context.Context, name string, args ...string) error {
	ec := getExecContext(ctx)
	cmd := newCommand(ctx, name, args...)
	cmd.Stdout = ec.out.Stdout
	cmd.Stderr = ec.out.Stderr
	// Always run from git root, or the specified path relative to it.
	if ec.path != "" {
		cmd.Dir = FromGitRoot(ec.path)
	} else {
		cmd.Dir = GitRoot()
	}
	return cmd.Run()
}

// ExecIn runs an external command in a specific directory.
// Use this when you need to run a command in a directory other than the current path.
//
// Example:
//
//	func updateDeps(ctx context.Context) error {
//	    return pocket.ExecIn(ctx, ".pocket", "go", "mod", "tidy")
//	}
func ExecIn(ctx context.Context, dir string, name string, args ...string) error {
	ec := getExecContext(ctx)
	cmd := newCommand(ctx, name, args...)
	cmd.Stdout = ec.out.Stdout
	cmd.Stderr = ec.out.Stderr
	cmd.Dir = dir
	return cmd.Run()
}

// newCommand creates a new command with .pocket/bin prepended to PATH.
func newCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	return commandBase(ctx, name, args...)
}

// Printf writes formatted output to stdout.
//
// Example:
//
//	pocket.Printf(ctx, "Processing %s...\n", path)
func Printf(ctx context.Context, format string, args ...any) {
	ec := getExecContext(ctx)
	fmt.Fprintf(ec.out.Stdout, format, args...)
}

// Println writes a line to stdout.
//
// Example:
//
//	pocket.Println(ctx, "Done")
func Println(ctx context.Context, args ...any) {
	ec := getExecContext(ctx)
	fmt.Fprintln(ec.out.Stdout, args...)
}

// Errorf writes formatted output to stderr.
func Errorf(ctx context.Context, format string, args ...any) {
	ec := getExecContext(ctx)
	fmt.Fprintf(ec.out.Stderr, format, args...)
}

// Options retrieves typed options from the context.
// Returns zero value if no options are set.
//
// Example:
//
//	type LintOptions struct {
//	    Config string
//	}
//
//	func goLint(ctx context.Context) error {
//	    opts := pocket.Options[LintOptions](ctx)
//	    return pocket.Exec(ctx, "golangci-lint", "run", "-c", opts.Config)
//	}
func Options[T any](ctx context.Context) T {
	ec := getExecContext(ctx)
	// Search through all stored options for one matching type T
	var zero T
	for _, opts := range ec.opts {
		if typed, ok := opts.(T); ok {
			return typed
		}
	}
	return zero
}

// Path returns the current execution path (relative to git root).
// Returns "." if no path is set.
//
// Example:
//
//	func formatTask(ctx context.Context) error {
//	    return pocket.Exec(ctx, "fmt", pocket.Path(ctx))
//	}
func Path(ctx context.Context) string {
	ec := getExecContext(ctx)
	if ec.path == "" {
		return "."
	}
	return ec.path
}

// CWD returns where the CLI was invoked (relative to git root).
func CWD(ctx context.Context) string {
	return getExecContext(ctx).cwd
}

// Verbose returns whether verbose mode is enabled.
func Verbose(ctx context.Context) bool {
	return getExecContext(ctx).verbose
}

// Output returns the current output writers.
// Use this when you need direct access to stdout/stderr writers.
func GetOutput(ctx context.Context) *Output {
	return getExecContext(ctx).out
}

// TaskContext provides runtime data for functions.
// This is a compatibility type for functions that need output writers.
type TaskContext struct {
	Path    string  // the path for this invocation (relative to git root)
	Verbose bool    // verbose mode
	Out     *Output // output writers
}

// Command creates an exec.Cmd with output wired to this context's output writers.
func (tc *TaskContext) Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := newCommand(ctx, name, args...)
	cmd.Stdout = tc.Out.Stdout
	cmd.Stderr = tc.Out.Stderr
	return cmd
}
