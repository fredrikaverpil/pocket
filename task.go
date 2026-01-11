package pocket

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// TaskAction is the function signature for task actions.
// ctx carries cancellation signals and deadlines.
// rc provides task-specific data: paths, options, output writers.
type TaskAction func(ctx context.Context, rc *RunContext) error

// RunContext provides all runtime state for task execution.
// It is passed explicitly through the Runnable chain.
type RunContext struct {
	// Public fields for task actions
	Paths   []string // resolved paths for this task (from Paths wrapper)
	Verbose bool     // verbose mode enabled
	Out     *Output  // output writers for stdout/stderr

	// Internal fields
	cwd           string                       // where CLI was invoked (relative to git root)
	parsedOptions any                          // typed options, access via GetOptions[T](rc)
	exec          *execution                   // tracks which tasks have run
	taskPaths     map[string][]string          // task name -> resolved paths
	taskArgs      map[string]map[string]string // task name -> CLI args
	skipRules     []skipRule                   // task skip rules
}

// NewRunContext creates a RunContext for a new execution.
func NewRunContext(out *Output, verbose bool, cwd string) *RunContext {
	return &RunContext{
		Out:       out,
		Verbose:   verbose,
		cwd:       cwd,
		exec:      newExecution(),
		taskPaths: make(map[string][]string),
		taskArgs:  make(map[string]map[string]string),
	}
}

// CWD returns the current working directory relative to git root.
func (rc *RunContext) CWD() string {
	return rc.cwd
}

// withOutput returns a copy with different output (for parallel buffering).
// Shares the same execution tracking.
func (rc *RunContext) withOutput(out *Output) *RunContext {
	cp := *rc
	cp.Out = out
	// Share execution (tracks what's done across all children)
	// Share maps (they're set up front, not modified during execution)
	return &cp
}

// withSkipRules returns a copy with additional skip rules.
func (rc *RunContext) withSkipRules(rules []skipRule) *RunContext {
	cp := *rc
	cp.skipRules = append(slices.Clone(rc.skipRules), rules...)
	return &cp
}

// setTaskPaths sets resolved paths for a task.
func (rc *RunContext) setTaskPaths(taskName string, paths []string) {
	rc.taskPaths[taskName] = paths
}

// SetTaskArgs sets CLI arguments for a task. This is used by the CLI
// to pass parsed command-line arguments to the task.
func (rc *RunContext) SetTaskArgs(taskName string, args map[string]string) {
	rc.taskArgs[taskName] = args
}

// ForEachPath executes fn for each path in the context.
// This is a convenience helper for the common pattern of iterating over paths.
// Iteration stops early if the context is cancelled (e.g., another parallel task failed).
func (rc *RunContext) ForEachPath(ctx context.Context, fn func(dir string) error) error {
	for _, dir := range rc.Paths {
		// Check for cancellation before each iteration.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := fn(dir); err != nil {
			return err
		}
	}
	return nil
}

// isSkipped checks if a task should be skipped for a given path.
func (rc *RunContext) isSkipped(taskName, path string) bool {
	for _, rule := range rc.skipRules {
		if rule.taskName != taskName {
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

// execution tracks which tasks have run in a single execution.
// This is shared across the entire Runnable tree.
type execution struct {
	mu     sync.Mutex
	done   map[string]bool
	errors map[string]error
}

func newExecution() *execution {
	return &execution{
		done:   make(map[string]bool),
		errors: make(map[string]error),
	}
}

func (e *execution) markDone(name string, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.done[name] = true
	if err != nil {
		e.errors[name] = err
	}
}

func (e *execution) isDone(name string) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.done[name] {
		return true, e.errors[name]
	}
	return false, nil
}

// skipRule defines a rule for skipping a task.
type skipRule struct {
	taskName string
	paths    []string // empty = skip everywhere, non-empty = skip only in these paths
}

// Task represents an immutable task definition.
//
// Create tasks using NewTask:
//
//	pocket.NewTask("my-task", "description", func(ctx context.Context, rc *pocket.RunContext) error {
//	    return nil
//	}).WithOptions(MyOptions{})
type Task struct {
	// Public fields (read-only after creation)
	Name    string
	Usage   string
	Options TaskOptions // typed options struct for CLI parsing (see args.go)
	Action  TaskAction  // function to execute when task runs
	Hidden  bool        // hide from CLI shim
	Builtin bool        // true for core tasks like generate, update, git-diff
}

// TaskName returns the task's CLI command name.
func (t *Task) TaskName() string {
	return t.Name
}

// NewTask creates an immutable task definition.
// Name is the CLI command name (e.g., "go-format").
// Usage is the help text shown in CLI.
// Action is the function executed when the task runs.
//
// Example:
//
//	pocket.NewTask("deploy", "deploy to environment", func(ctx context.Context, rc *pocket.RunContext) error {
//	    opts := pocket.GetOptions[DeployOptions](rc)
//	    return deploy(ctx, opts.Env)
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

// WithOptions returns a new Task with typed options for CLI flag parsing.
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
	cp := *t
	cp.Options = opts
	return &cp
}

// AsHidden returns a new Task marked as hidden from CLI help output.
// Hidden tasks can still be run directly by name.
func (t *Task) AsHidden() *Task {
	cp := *t
	cp.Hidden = true
	return &cp
}

// AsBuiltin returns a new Task marked as a built-in task.
// This is used internally for core tasks like generate, update, git-diff.
func (t *Task) AsBuiltin() *Task {
	cp := *t
	cp.Builtin = true
	return &cp
}

// Run executes the task's action exactly once per execution.
// Implements the Runnable interface.
// Skip rules from RunContext are checked:
// - Global skip (no paths): task doesn't run at all
// - Path-specific skip: those paths are filtered from execution.
func (t *Task) Run(ctx context.Context, rc *RunContext) error {
	// Check if already done in this execution.
	if done, err := rc.exec.isDone(t.Name); done {
		return err
	}

	// Check for global skip (rule with no paths).
	if rc.isSkipped(t.Name, "") {
		rc.exec.markDone(t.Name, nil)
		return nil
	}

	// Determine paths, defaulting to cwd if not set.
	paths := rc.taskPaths[t.Name]
	if len(paths) == 0 {
		paths = []string{rc.cwd}
	}

	// Filter out paths that should be skipped.
	var filteredPaths []string
	var skippedPaths []string
	for _, p := range paths {
		if !rc.isSkipped(t.Name, p) {
			filteredPaths = append(filteredPaths, p)
		} else {
			skippedPaths = append(skippedPaths, p)
		}
	}

	// If all paths are skipped, don't run.
	if len(filteredPaths) == 0 {
		fmt.Fprintf(rc.Out.Stdout, "=== %s (skipped)\n", t.Name)
		rc.exec.markDone(t.Name, nil)
		return nil
	}

	// Show task name with any skipped paths.
	if len(skippedPaths) > 0 {
		fmt.Fprintf(rc.Out.Stdout, "=== %s (skipped in: %s)\n", t.Name, strings.Join(skippedPaths, ", "))
	} else {
		fmt.Fprintf(rc.Out.Stdout, "=== %s\n", t.Name)
	}

	// Parse typed options (merge defaults from t.Options with CLI overrides).
	args := rc.taskArgs[t.Name]
	parsedOptions, err := parseOptionsFromCLI(t.Options, args)
	if err != nil {
		rc.exec.markDone(t.Name, fmt.Errorf("parse options: %w", err))
		return err
	}

	// Build task-specific RunContext.
	taskRC := &RunContext{
		Paths:         filteredPaths,
		Verbose:       rc.Verbose,
		Out:           rc.Out,
		cwd:           rc.cwd,
		parsedOptions: parsedOptions,
		exec:          rc.exec,
		taskPaths:     rc.taskPaths,
		taskArgs:      rc.taskArgs,
		skipRules:     rc.skipRules,
	}

	// Run the action.
	err = t.Action(ctx, taskRC)
	rc.exec.markDone(t.Name, err)
	return err
}

// Tasks returns this task as a slice (implements Runnable interface).
func (t *Task) Tasks() []*Task {
	return []*Task{t}
}
