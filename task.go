package pocket

import (
	"context"
	"fmt"
	"maps"
	"sync"
)

// ArgDef defines an argument that a task accepts.
type ArgDef struct {
	Name    string // argument name (used as key in key=value)
	Usage   string // description for help output
	Default string // default value if not provided
}

// RunContext provides runtime context to Actions.
type RunContext struct {
	Args  map[string]string // CLI arguments (key=value pairs)
	Paths []string          // resolved paths for this task (from Paths wrapper)
	Cwd   string            // current working directory (relative to git root)
}

// Task represents a runnable task.
type Task struct {
	Name    string
	Usage   string
	Args    []ArgDef // declared arguments this task accepts
	Action  func(ctx context.Context, rc *RunContext) error
	Hidden  bool
	Builtin bool // true for core tasks like generate, update, git-diff

	// once ensures the task runs exactly once per execution.
	once sync.Once
	// err stores the result of the task execution.
	err error
	// args stores the parsed arguments for this execution.
	args map[string]string
	// paths stores the resolved paths for this execution.
	paths []string
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

// SetPaths sets the resolved paths for this task execution.
func (t *Task) SetPaths(paths []string) {
	t.paths = paths
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
		// Build RunContext with resolved paths and cwd.
		rc := &RunContext{
			Args:  t.args,
			Paths: t.paths,
			Cwd:   CwdFromContext(ctx),
		}
		// Default to cwd if no paths were set.
		if len(rc.Paths) == 0 {
			rc.Paths = []string{rc.Cwd}
		}
		t.err = t.Action(ctx, rc)
	})
	return t.err
}

// Tasks returns this task as a slice (implements Runnable interface).
func (t *Task) Tasks() []*Task {
	return []*Task{t}
}
