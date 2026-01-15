package pocket

import (
	"context"
	"fmt"
	"sync"
)

// execContext holds runtime state for function execution.
// Stored in context.Context and accessed via helper functions.
type execContext struct {
	mode    execMode       // execution mode (execute or collect)
	plan    *ExecutionPlan // plan being collected (only in modeCollect)
	out     *Output        // where to write output
	path    string         // current path for this invocation
	cwd     string         // where CLI was invoked (relative to git root)
	verbose bool           // verbose mode enabled
	opts    map[string]any // func name -> options struct
	dedup   *dedupState    // shared deduplication state (thread-safe)
}

// dedupState tracks executed runnables for deduplication.
// Shared across parallel executions with thread-safe access.
type dedupState struct {
	mu       sync.Mutex
	executed map[uintptr]bool
}

// newDedupState creates a new deduplication state.
func newDedupState() *dedupState {
	return &dedupState{
		executed: make(map[uintptr]bool),
	}
}

// shouldRun checks if a runnable should run (not already executed).
// Marks it as executed if it should run. Thread-safe.
func (d *dedupState) shouldRun(key uintptr) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.executed[key] {
		return false
	}
	d.executed[key] = true
	return true
}

// contextKey is the type for context keys to avoid collisions.
type contextKey int

const (
	execContextKey contextKey = iota
)

// newExecContext creates a new execution context.
func newExecContext(out *Output, cwd string, verbose bool) *execContext {
	return &execContext{
		mode:    modeExecute, // explicit for clarity (default is execute)
		out:     out,
		cwd:     cwd,
		verbose: verbose,
		opts:    make(map[string]any),
		dedup:   newDedupState(),
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
	// In collect mode, don't execute - just return success
	if ec.mode == modeCollect {
		return nil
	}
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
func ExecIn(ctx context.Context, dir, name string, args ...string) error {
	ec := getExecContext(ctx)
	// In collect mode, don't execute - just return success
	if ec.mode == modeCollect {
		return nil
	}
	cmd := newCommand(ctx, name, args...)
	cmd.Stdout = ec.out.Stdout
	cmd.Stderr = ec.out.Stderr
	cmd.Dir = dir
	return cmd.Run()
}

// Printf writes formatted output to stdout.
// In collect mode, this is a no-op.
//
// Example:
//
//	pocket.Printf(ctx, "Processing %s...\n", path)
func Printf(ctx context.Context, format string, args ...any) {
	ec := getExecContext(ctx)
	if ec.mode == modeCollect {
		return
	}
	fmt.Fprintf(ec.out.Stdout, format, args...)
}

// Println writes a line to stdout.
// In collect mode, this is a no-op.
//
// Example:
//
//	pocket.Println(ctx, "Done")
func Println(ctx context.Context, args ...any) {
	ec := getExecContext(ctx)
	if ec.mode == modeCollect {
		return
	}
	fmt.Fprintln(ec.out.Stdout, args...)
}

// printTaskHeader writes the task execution header to output.
// Format: ":: task-name" or ":: task-name [path]" when running in a specific path.
func printTaskHeader(ctx context.Context, name string) {
	ec := getExecContext(ctx)
	if ec.path != "" && ec.path != "." {
		fmt.Fprintf(ec.out.Stdout, ":: %s [%s]\n", name, ec.path)
	} else {
		fmt.Fprintf(ec.out.Stdout, ":: %s\n", name)
	}
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

// Verbose returns whether verbose mode is enabled.
func Verbose(ctx context.Context) bool {
	return getExecContext(ctx).verbose
}

// CWD returns where the CLI was invoked (relative to git root).
func CWD(ctx context.Context) string {
	return getExecContext(ctx).cwd
}

// GetOutput returns the current output writers.
// Use this when you need direct access to stdout/stderr writers.
func GetOutput(ctx context.Context) *Output {
	return getExecContext(ctx).out
}

// Errorf writes formatted output to stderr.
// In collect mode, this is a no-op.
func Errorf(ctx context.Context, format string, args ...any) {
	ec := getExecContext(ctx)
	if ec.mode == modeCollect {
		return
	}
	fmt.Fprintf(ec.out.Stderr, format, args...)
}

// TestContext creates a context suitable for testing.
// It sets up an execContext with the given output and working directory ".".
//
// Example:
//
//	func TestMyTask(t *testing.T) {
//	    out := pocket.StdOutput()
//	    ctx := pocket.TestContext(out)
//	    if err := myTask(ctx); err != nil {
//	        t.Fatal(err)
//	    }
//	}
func TestContext(out *Output) context.Context {
	ec := newExecContext(out, ".", false)
	return withExecContext(context.Background(), ec)
}
