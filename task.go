package pocket

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// TaskAction is the function signature for task actions.
// Actions receive RunContext (which provides context via rc.Context()) and return an error if the task fails.
type TaskAction func(rc *RunContext) error

// RunContext provides runtime context to Actions.
type RunContext struct {
	Paths   []string // resolved paths for this task (from Paths wrapper)
	Verbose bool     // verbose mode enabled

	ctx           context.Context // internal: for cancellation checks
	cwd           string          // internal: where CLI was invoked (relative to git root)
	parsedOptions any             // typed options, access via GetOptions[T](rc)
	skipRules     []skipRule      // internal: task skip rules
}

// Context returns the context for this task execution.
// Use this to check for cancellation or pass to sub-operations.
func (rc *RunContext) Context() context.Context {
	if rc.ctx == nil {
		return context.Background()
	}
	return rc.ctx
}

// ForEachPath executes fn for each path in the context.
// This is a convenience helper for the common pattern of iterating over paths.
// Iteration stops early if the context is cancelled (e.g., another parallel task failed).
func (rc *RunContext) ForEachPath(fn func(dir string) error) error {
	for _, dir := range rc.Paths {
		// Check for cancellation before each iteration.
		select {
		case <-rc.Context().Done():
			return rc.Context().Err()
		default:
		}
		if err := fn(dir); err != nil {
			return err
		}
	}
	return nil
}

// Task represents a runnable task.
//
// Create tasks using NewTask:
//
//	pocket.NewTask("my-task", "description", func(rc *pocket.RunContext) error {
//	    return nil
//	}).WithOptions(MyOptions{})
type Task struct {
	// Public fields (for backwards compatibility during migration)
	//
	// Deprecated: Use NewTask() constructor instead of struct literals.
	Name    string
	Usage   string
	Options TaskOptions // typed options struct for CLI parsing (see args.go)
	Action  TaskAction  // function to execute when task runs
	Hidden  bool        // hide from CLI shim
	Builtin bool        // true for core tasks like generate, update, git-diff

	// once ensures the task runs exactly once per execution.
	once sync.Once
	// err stores the result of the task execution.
	err error
	// cliArgs stores the CLI arguments for this execution.
	cliArgs map[string]string
	// paths stores the resolved paths for this execution.
	paths []string
}

// TaskName returns the task's CLI command name.
func (t *Task) TaskName() string {
	return t.Name
}

// NewTask creates a task with the required fields.
// Name is the CLI command name (e.g., "go-format").
// Usage is the help text shown in CLI.
// Action is the function executed when the task runs.
//
// Example:
//
//	pocket.NewTask("deploy", "deploy to environment", func(rc *pocket.RunContext) error {
//	    opts := pocket.GetOptions[DeployOptions](rc)
//	    return deploy(opts.Env)
//	}).WithOptions(DeployOptions{Env: "staging"})
func NewTask(name, usage string, action TaskAction) *Task {
	if name == "" {
		panic("pocket.NewTask: name is required")
	}
	if usage == "" {
		panic("pocket.NewTask: usage is required")
	}
	if action == nil {
		panic("pocket.NewTask: action is required")
	}
	return &Task{
		Name:   name,
		Usage:  usage,
		Action: action,
	}
}

// WithOptions sets typed options for CLI flag parsing.
// Options must be a struct with exported fields of type bool, string, or int.
// Use struct tags to customize: `usage:"help text"` and `arg:"flag-name"`.
//
// Example:
//
//	type DeployOptions struct {
//	    Env    string `usage:"target environment"`
//	    DryRun bool   `usage:"print without executing"`
//	}
//
//	NewTask("deploy", "deploy app", deployAction).
//	    WithOptions(DeployOptions{Env: "staging"})
func (t *Task) WithOptions(opts any) *Task {
	if opts != nil {
		if _, err := inspectArgs(opts); err != nil {
			panic(fmt.Sprintf("pocket.Task.WithOptions: %v", err))
		}
	}
	t.Options = opts
	return t
}

// AsHidden marks the task as hidden from CLI help output.
// Hidden tasks can still be run directly by name.
func (t *Task) AsHidden() *Task {
	t.Hidden = true
	return t
}

// AsBuiltin marks the task as a built-in task.
// This is used internally for core tasks like generate, update, git-diff.
func (t *Task) AsBuiltin() *Task {
	t.Builtin = true
	return t
}

// contextKey is the type for context keys used by this package.
type contextKey int

const runContextKey contextKey = 0

// skipRule defines a rule for skipping a task.
type skipRule struct {
	taskName string
	paths    []string // empty = skip everywhere, non-empty = skip only in these paths
}

// getRunContext returns the RunContext from context.
func getRunContext(ctx context.Context) *RunContext {
	if rc, ok := ctx.Value(runContextKey).(*RunContext); ok {
		return rc
	}
	return &RunContext{cwd: "."}
}

// withRunContext returns a context with the RunContext set.
func withRunContext(ctx context.Context, rc *RunContext) context.Context {
	return context.WithValue(ctx, runContextKey, rc)
}

// withSkipRules returns a new context with skip rules added.
func withSkipRules(ctx context.Context, rules []skipRule) context.Context {
	rc := getRunContext(ctx)
	return withRunContext(ctx, &RunContext{
		Verbose:   rc.Verbose,
		cwd:       rc.cwd,
		skipRules: rules,
	})
}

// isSkipped returns true if the task should be skipped for the given path.
func isSkipped(ctx context.Context, name, path string) bool {
	rc := getRunContext(ctx)
	for _, rule := range rc.skipRules {
		if rule.taskName != name {
			continue
		}
		if len(rule.paths) == 0 {
			return true // global skip
		}
		if slices.Contains(rule.paths, path) {
			return true // path-specific skip
		}
	}
	return false
}

// SetArgs sets the CLI arguments for this task execution.
// These will be merged with defaults from Options struct when the task runs.
func (t *Task) SetArgs(args map[string]string) {
	t.cliArgs = args
}

// SetPaths sets the resolved paths for this task execution.
func (t *Task) SetPaths(paths []string) {
	t.paths = paths
}

// Run executes the task's action exactly once.
// Implements the Runnable interface.
// Skip rules from context are checked:
// - Global skip (no paths): task doesn't run at all
// - Path-specific skip: those paths are filtered from execution.
func (t *Task) Run(ctx context.Context) error {
	// Check for global skip (rule with no paths).
	if isSkipped(ctx, t.Name, "") {
		return nil
	}
	t.once.Do(func() {
		if t.Action == nil {
			return
		}
		base := getRunContext(ctx)
		// Determine paths, defaulting to cwd if not set.
		paths := t.paths
		if len(paths) == 0 {
			paths = []string{base.cwd}
		}
		// Filter out paths that should be skipped.
		var filteredPaths []string
		var skippedPaths []string
		for _, p := range paths {
			if !isSkipped(ctx, t.Name, p) {
				filteredPaths = append(filteredPaths, p)
			} else {
				skippedPaths = append(skippedPaths, p)
			}
		}
		// If all paths are skipped, don't run.
		if len(filteredPaths) == 0 {
			fmt.Fprintf(Stdout(ctx), "=== %s (skipped)\n", t.Name)
			return
		}
		// Show task name with any skipped paths.
		if len(skippedPaths) > 0 {
			fmt.Fprintf(Stdout(ctx), "=== %s (skipped in: %s)\n", t.Name, strings.Join(skippedPaths, ", "))
		} else {
			fmt.Fprintf(Stdout(ctx), "=== %s\n", t.Name)
		}
		// Parse typed options (merge defaults from t.Options with CLI overrides).
		parsedOptions, err := parseOptionsFromCLI(t.Options, t.cliArgs)
		if err != nil {
			t.err = fmt.Errorf("parse options: %w", err)
			return
		}
		// Build RunContext for this task.
		rc := &RunContext{
			Paths:         filteredPaths,
			Verbose:       base.Verbose,
			ctx:           ctx,
			cwd:           base.cwd,
			parsedOptions: parsedOptions,
		}
		t.err = t.Action(rc)
	})
	return t.err
}

// Tasks returns this task as a slice (implements Runnable interface).
func (t *Task) Tasks() []*Task {
	return []*Task{t}
}
