package pocket

import (
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"sync"
)

// TaskAction is the function signature for task actions.
// ctx carries cancellation signals and deadlines.
// tc provides task-specific data: paths, options, output writers.
type TaskAction func(ctx context.Context, tc *TaskContext) error

// Execution holds state shared across the entire Runnable tree.
// It is created once by the CLI and passed through all Runnables.
type Execution struct {
	// Output writers (can be swapped for buffering in parallel execution)
	Out *Output

	// Internal state (shared across execution tree)
	state *executionState // cwd, verbose, dedup (immutable after creation)
	setup *taskSetup      // paths, args, skipRules (accumulated during traversal)
}

// NewExecution creates an Execution for a new run.
func NewExecution(out *Output, verbose bool, cwd string) *Execution {
	return &Execution{
		Out:   out,
		state: newExecutionState(cwd, verbose),
		setup: newTaskSetup(),
	}
}

// CWD returns the current working directory relative to git root.
func (e *Execution) CWD() string {
	return e.state.cwd
}

// Verbose returns whether verbose mode is enabled.
func (e *Execution) Verbose() bool {
	return e.state.verbose
}

// withOutput returns a copy with different output (for parallel buffering).
// Shares the same execution tracking and setup.
func (e *Execution) withOutput(out *Output) *Execution {
	return &Execution{
		Out:   out,
		state: e.state, // shared
		setup: e.setup, // shared
	}
}

// withSkipRules returns a copy with additional skip rules.
func (e *Execution) withSkipRules(rules []skipRule) *Execution {
	return &Execution{
		Out:   e.Out,
		state: e.state,
		setup: e.setup.withSkipRules(rules),
	}
}

// setTaskPaths sets resolved paths for a task.
func (e *Execution) setTaskPaths(taskName string, paths []string) {
	e.setup.paths[taskName] = paths
}

// SetTaskArgs sets CLI arguments for a task. This is used by the CLI
// to pass parsed command-line arguments to the task.
func (e *Execution) SetTaskArgs(taskName string, args map[string]string) {
	e.setup.args[taskName] = args
}

// isSkipped checks if a task should be skipped for a given path.
func (e *Execution) isSkipped(taskName, path string) bool {
	for _, rule := range e.setup.skipRules {
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

// shouldSkipGlobally checks if a task should be skipped entirely (global skip rule).
func (e *Execution) shouldSkipGlobally(taskName string) bool {
	return e.isSkipped(taskName, "")
}

// resolveAndFilterPaths returns the paths for a task after applying skip filters.
// Returns the paths to run and the paths that were skipped.
func (e *Execution) resolveAndFilterPaths(taskName string) (paths, skipped []string) {
	// Determine paths, defaulting to cwd if not set.
	all := e.setup.paths[taskName]
	if len(all) == 0 {
		all = []string{e.state.cwd}
	}

	// Filter out paths that should be skipped.
	for _, p := range all {
		if !e.isSkipped(taskName, p) {
			paths = append(paths, p)
		} else {
			skipped = append(skipped, p)
		}
	}
	return paths, skipped
}

// printTaskHeader writes the task execution header to output.
func (e *Execution) printTaskHeader(taskName string, skippedPaths []string) {
	if len(skippedPaths) > 0 {
		fmt.Fprintf(e.Out.Stdout, ":: %s (skipped in: %s)\n", taskName, strings.Join(skippedPaths, ", "))
	} else {
		fmt.Fprintf(e.Out.Stdout, ":: %s\n", taskName)
	}
}

// TaskContext provides runtime data for task actions.
// It is created by Task.Run() with resolved paths and options.
type TaskContext struct {
	// Task-specific data
	Paths   []string // resolved paths for this task
	Verbose bool     // verbose mode (copied for convenience)
	Out     *Output  // output writers

	// Parsed options (access via GetOptions[T])
	options any

	// Reference to execution for CWD access
	exec *Execution
}

// CWD returns the current working directory relative to git root.
func (tc *TaskContext) CWD() string {
	return tc.exec.CWD()
}

// Execution returns the underlying Execution for running other Runnables.
// This is useful when a task action needs to orchestrate other tasks.
func (tc *TaskContext) Execution() *Execution {
	return tc.exec
}

// ForEachPath executes fn for each path in the context.
// This is a convenience helper for the common pattern of iterating over paths.
// Iteration stops early if the context is cancelled (e.g., another parallel task failed).
func (tc *TaskContext) ForEachPath(ctx context.Context, fn func(dir string) error) error {
	for _, dir := range tc.Paths {
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

// Command creates an exec.Cmd with output wired to this task's output writers.
// This ensures command output is properly buffered when running in parallel.
//
// The command has:
//   - PATH prepended with .pocket/bin/
//   - Color-forcing environment variables (when stdout is a TTY)
//   - Graceful shutdown on context cancellation
//   - Stdout/Stderr connected to tc.Out (for proper parallel buffering)
//
// Example:
//
//	cmd := tc.Command(ctx, "go", "test", "./...")
//	cmd.Dir = pocket.FromGitRoot(dir)
//	return cmd.Run()
func (tc *TaskContext) Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := commandBase(ctx, name, args...)
	cmd.Stdout = tc.Out.Stdout
	cmd.Stderr = tc.Out.Stderr
	return cmd
}

// Tooler is an interface for tools that can create commands.
// This matches the *tool.Tool type from the tool package.
type Tooler interface {
	// Command prepares the tool and returns an exec.Cmd for running it.
	Command(ctx context.Context, args ...string) (*exec.Cmd, error)
}

// Tool returns a ToolRunner that wraps a tool with output wiring.
// Commands created through the ToolRunner have their output automatically
// connected to this task's output writers for proper parallel buffering.
//
// Example:
//
//	cmd, err := tc.Tool(golangcilint.T).Command(ctx, "run", "./...")
//	if err != nil {
//	    return err
//	}
//	cmd.Dir = pocket.FromGitRoot(dir)
//	return cmd.Run()
func (tc *TaskContext) Tool(t Tooler) *ToolRunner {
	return &ToolRunner{tool: t, out: tc.Out}
}

// ToolRunner wraps a tool with output wiring.
// It ensures commands created from the tool have their output connected
// to the task's output writers.
type ToolRunner struct {
	tool Tooler
	out  *Output
}

// Command prepares the tool and returns an exec.Cmd with output wired.
func (tr *ToolRunner) Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
	cmd, err := tr.tool.Command(ctx, args...)
	if err != nil {
		return nil, err
	}
	cmd.Stdout = tr.out.Stdout
	cmd.Stderr = tr.out.Stderr
	return cmd, nil
}

// Run prepares and executes the tool with output wired.
func (tr *ToolRunner) Run(ctx context.Context, args ...string) error {
	cmd, err := tr.Command(ctx, args...)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// dedupTracker tracks which tasks have run in a single execution.
// This is shared across the entire Runnable tree.
type dedupTracker struct {
	mu     sync.Mutex
	done   map[string]bool
	errors map[string]error
}

func newDedupTracker() *dedupTracker {
	return &dedupTracker{
		done:   make(map[string]bool),
		errors: make(map[string]error),
	}
}

func (e *dedupTracker) markDone(name string, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.done[name] = true
	if err != nil {
		e.errors[name] = err
	}
}

func (e *dedupTracker) isDone(name string) (bool, error) {
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
//	pocket.NewTask("my-task", "description", func(ctx context.Context, tc *pocket.TaskContext) error {
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
//	pocket.NewTask("deploy", "deploy to environment", func(ctx context.Context, tc *pocket.TaskContext) error {
//	    opts := pocket.GetOptions[DeployOptions](tc)
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
// Skip rules from Execution are checked:
// - Global skip (no paths): task doesn't run at all
// - Path-specific skip: those paths are filtered from execution.
func (t *Task) Run(ctx context.Context, exec *Execution) error {
	dedup := exec.state.dedup

	// Check if already done in this execution.
	if done, err := dedup.isDone(t.Name); done {
		return err
	}

	// Check for global skip.
	if exec.shouldSkipGlobally(t.Name) {
		dedup.markDone(t.Name, nil)
		return nil
	}

	// Resolve paths and filter skipped ones.
	paths, skipped := exec.resolveAndFilterPaths(t.Name)
	if len(paths) == 0 {
		fmt.Fprintf(exec.Out.Stdout, ":: %s (skipped)\n", t.Name)
		dedup.markDone(t.Name, nil)
		return nil
	}

	// Print task header.
	exec.printTaskHeader(t.Name, skipped)

	// Parse typed options.
	opts, err := parseOptionsFromCLI(t.Options, exec.setup.args[t.Name])
	if err != nil {
		dedup.markDone(t.Name, fmt.Errorf("parse options: %w", err))
		return err
	}

	// Build TaskContext and run the action.
	tc := &TaskContext{
		Paths:   paths,
		Verbose: exec.Verbose(),
		Out:     exec.Out,
		options: opts,
		exec:    exec,
	}
	err = t.Action(ctx, tc)
	dedup.markDone(t.Name, err)
	return err
}

// Tasks returns this task as a slice (implements Runnable interface).
func (t *Task) Tasks() []*Task {
	return []*Task{t}
}
