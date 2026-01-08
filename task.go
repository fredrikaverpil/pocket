package pocket

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"golang.org/x/sync/errgroup"
)

// ArgDef defines an argument that a task accepts.
type ArgDef struct {
	Name    string // argument name (used as key in key=value)
	Usage   string // description for help output
	Default string // default value if not provided
}

// Task represents a runnable task.
type Task struct {
	Name    string
	Usage   string
	Args    []ArgDef // declared arguments this task accepts
	Action  func(ctx context.Context, args map[string]string) error
	Hidden  bool
	Builtin bool // true for core tasks like generate, update, git-diff

	// once ensures the task runs exactly once per execution.
	once sync.Once
	// err stores the result of the task execution.
	err error
	// args stores the parsed arguments for this execution.
	args map[string]string
}

// contextKey is the type for context keys used by this package.
type contextKey int

const (
	// verboseKey is the context key for verbose mode.
	verboseKey contextKey = iota
	// cwdKey is the context key for current working directory (relative to git root).
	cwdKey
)

// WithVerbose returns a context with verbose mode set.
func WithVerbose(ctx context.Context, verbose bool) context.Context {
	return context.WithValue(ctx, verboseKey, verbose)
}

// IsVerbose returns true if verbose mode is enabled in the context.
func IsVerbose(ctx context.Context) bool {
	v, _ := ctx.Value(verboseKey).(bool)
	return v
}

// WithCwd returns a context with the current working directory set.
// The cwd should be relative to git root.
func WithCwd(ctx context.Context, cwd string) context.Context {
	return context.WithValue(ctx, cwdKey, cwd)
}

// CwdFromContext returns the current working directory from context.
// Returns "." if not set.
func CwdFromContext(ctx context.Context) string {
	if cwd, ok := ctx.Value(cwdKey).(string); ok {
		return cwd
	}
	return "."
}

// SetArgs sets the arguments for this task execution.
// Arguments are merged with defaults defined in Args.
func (t *Task) SetArgs(args map[string]string) {
	t.args = make(map[string]string)
	// Apply defaults first.
	for _, arg := range t.Args {
		if arg.Default != "" {
			t.args[arg.Name] = arg.Default
		}
	}
	// Override with provided args.
	maps.Copy(t.args, args)
}

// Run executes the task's action exactly once.
// Implements the Runnable interface.
func (t *Task) Run(ctx context.Context) error {
	t.once.Do(func() {
		if t.Action == nil {
			return
		}
		// Always show task name for progress feedback.
		fmt.Printf("=== %s\n", t.Name)
		// Ensure args map exists (may be nil if SetArgs wasn't called).
		if t.args == nil {
			t.SetArgs(nil)
		}
		t.err = t.Action(ctx, t.args)
	})
	return t.err
}

// Tasks returns this task as a slice (implements Runnable interface).
func (t *Task) Tasks() []*Task {
	return []*Task{t}
}

// Deps runs the given tasks in parallel and waits for all to complete.
// Each task runs at most once regardless of how many times Deps is called.
// If any task fails, Deps returns the first error encountered.
func Deps(ctx context.Context, tasks ...*Task) error {
	if len(tasks) == 0 {
		return nil
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, task := range tasks {
		if task == nil {
			continue
		}
		g.Go(func() error {
			return task.Run(ctx)
		})
	}
	return g.Wait()
}

// SerialDeps runs the given tasks sequentially in order.
// Each task runs at most once regardless of how many times SerialDeps is called.
// If any task fails, SerialDeps returns immediately with the error.
func SerialDeps(ctx context.Context, tasks ...*Task) error {
	for _, task := range tasks {
		if task == nil {
			continue
		}
		if err := task.Run(ctx); err != nil {
			return err
		}
	}
	return nil
}
