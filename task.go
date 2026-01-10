package pocket

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// RunContext provides runtime context to Actions.
type RunContext struct {
	Paths   []string // resolved paths for this task (from Paths wrapper)
	Cwd     string   // current working directory (relative to git root)
	Verbose bool     // verbose mode enabled

	parsedArgs any        // typed args, access via GetArgs[T](rc)
	skipRules  []skipRule // internal: task skip rules
}

// ForEachPath executes fn for each path in the context.
// This is a convenience helper for the common pattern of iterating over paths.
func (rc *RunContext) ForEachPath(fn func(dir string) error) error {
	for _, dir := range rc.Paths {
		if err := fn(dir); err != nil {
			return err
		}
	}
	return nil
}

// Task represents a runnable task.
type Task struct {
	Name    string
	Usage   string
	Args    any // typed args struct, inspected via reflection for CLI parsing
	Action  func(ctx context.Context, rc *RunContext) error
	Hidden  bool
	Builtin bool // true for core tasks like generate, update, git-diff

	// once ensures the task runs exactly once per execution.
	once sync.Once
	// err stores the result of the task execution.
	err error
	// cliArgs stores the CLI arguments for this execution.
	cliArgs map[string]string
	// paths stores the resolved paths for this execution.
	paths []string
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
	return &RunContext{Cwd: "."}
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
		Cwd:       rc.Cwd,
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
// These will be merged with defaults from Args struct when the task runs.
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
			paths = []string{base.Cwd}
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
		// Parse typed args (merge defaults from t.Args with CLI overrides).
		parsedArgs, err := parseArgsFromCLI(t.Args, t.cliArgs)
		if err != nil {
			t.err = fmt.Errorf("parse args: %w", err)
			return
		}
		// Build RunContext for this task.
		rc := &RunContext{
			Paths:      filteredPaths,
			Cwd:        base.Cwd,
			Verbose:    base.Verbose,
			parsedArgs: parsedArgs,
		}
		t.err = t.Action(ctx, rc)
	})
	return t.err
}

// Tasks returns this task as a slice (implements Runnable interface).
func (t *Task) Tasks() []*Task {
	return []*Task{t}
}
