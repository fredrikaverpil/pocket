package pocket

import (
	"context"
)

// TaskDef represents a named function that can be executed.
// Create with pocket.Task() - this is the only way to create runnable functions.
//
// The body can be:
//   - pocket.Run(name, args...) - static command
//   - pocket.Do(fn) - dynamic commands or arbitrary Go code
//   - pocket.Serial(...) or pocket.Parallel(...) - compositions
//
// Example:
//
//	// Simple: static command
//	var Format = pocket.Task("go-format", "format Go code",
//	    pocket.Run("go", "fmt", "./..."),
//	)
//
//	// Composed: install dependency then run
//	var Lint = pocket.Task("go-lint", "run linter", pocket.Serial(
//	    InstallLinter,
//	    pocket.Run("golangci-lint", "run", "./..."),
//	))
//
//	// Dynamic: args computed at runtime
//	var Test = pocket.Task("go-test", "run tests", testCmd())
//	func testCmd() pocket.Runnable {
//	    return pocket.Do(func(ctx context.Context) error {
//	        args := []string{"test"}
//	        if pocket.Verbose(ctx) {
//	            args = append(args, "-v")
//	        }
//	        return pocket.Exec(ctx, "go", append(args, "./...")...)
//	    })
//	}
//
//	// Hidden: tool installers
//	var InstallLinter = pocket.Task("install:linter", "install linter",
//	    pocket.InstallGo("github.com/org/linter", "v1.0.0"),
//	).Hidden()
type TaskDef struct {
	name   string
	usage  string
	body   Runnable
	opts   any
	hidden bool
	silent bool // suppress task header output (for machine-readable output)
}

// TaskOpt configures a task created with Task().
type TaskOpt func(*TaskDef)

// Task creates a new named task.
// This is the primary way to create tasks that appear in the CLI.
//
// The name is used for CLI commands (e.g., "go-format" becomes ./pok go-format).
// The usage is displayed in help output.
// The body can be:
//   - Runnable - from Run, Do, Serial, Parallel
//   - func(context.Context) error - legacy, wrapped automatically
//
// Use options to configure the task:
//
//	var Lint = pocket.Task("lint", "run linter", lintCmd(),
//	    pocket.Opts(LintOptions{}),
//	    pocket.AsHidden(),
//	)
func Task(name, usage string, body any, opts ...TaskOpt) *TaskDef {
	if name == "" {
		panic("pocket.Func: name is required")
	}
	if usage == "" {
		panic("pocket.Func: usage is required")
	}
	if body == nil {
		panic("pocket.Func: body is required")
	}

	td := &TaskDef{
		name:  name,
		usage: usage,
		body:  toRunnable(body),
	}
	for _, opt := range opts {
		opt(td)
	}
	return td
}

// Opts attaches CLI options to a task.
// Options are accessible via pocket.Options[T](ctx) in the function.
//
// Example:
//
//	type FormatOptions struct {
//	    Config string `arg:"config" usage:"path to config file"`
//	}
//
//	var Format = pocket.Task("format", "format code", formatImpl,
//	    pocket.Opts(FormatOptions{Config: ".golangci.yml"}),
//	)
func Opts(opts any) TaskOpt {
	return func(td *TaskDef) {
		td.opts = opts
	}
}

// AsHidden marks a task as hidden from CLI help.
// Hidden tasks can still be executed but don't appear in ./pok -h.
// Use this for internal tasks like tool installers.
//
// Example:
//
//	var Install = pocket.Task("install:tool", "install tool",
//	    pocket.InstallGo("github.com/org/tool", "v1.0.0"),
//	    pocket.AsHidden(),
//	)
func AsHidden() TaskOpt {
	return func(td *TaskDef) {
		td.hidden = true
	}
}

// AsSilent suppresses the task header output (e.g., ":: task-name").
// Use this for tasks that produce machine-readable output (JSON, etc.).
//
// Example:
//
//	var Matrix = pocket.Task("gha-matrix", "output GHA matrix JSON",
//	    matrixCmd(),
//	    pocket.AsSilent(),
//	)
func AsSilent() TaskOpt {
	return func(td *TaskDef) {
		td.silent = true
	}
}

// Named sets a different CLI name for the task.
// Use this when the same task needs different names in different contexts.
//
// Example:
//
//	pocket.Task("integration-test", "run integration tests", testImpl,
//	    pocket.Named("integration-test"),
//	)
func Named(name string) TaskOpt {
	if name == "" {
		panic("pocket.Named: name is required")
	}
	return func(td *TaskDef) {
		td.name = name
	}
}

// Usage sets different help text for the task.
//
// Example:
//
//	pocket.Task("test", "run tests", testImpl,
//	    pocket.Usage("run integration tests"),
//	)
func Usage(usage string) TaskOpt {
	if usage == "" {
		panic("pocket.Usage: usage is required")
	}
	return func(td *TaskDef) {
		td.usage = usage
	}
}

// Name returns the function's CLI name.
func (f *TaskDef) Name() string {
	return f.name
}

// Usage returns the function's help text.
func (f *TaskDef) Usage() string {
	return f.usage
}

// IsHidden returns whether the function is hidden from CLI help.
func (f *TaskDef) IsHidden() bool {
	return f.hidden
}

// GetOpts returns the function's options, or nil if none.
func (f *TaskDef) GetOpts() any {
	return f.opts
}

// WithOpts creates a copy of a task with different options.
// This is useful for creating task variants at runtime, such as
// applying CLI-parsed options or package-level configuration.
//
// Example:
//
//	// In a task package's Tasks() function:
//	lintTask := Lint
//	if cfg.lint != (LintOptions{}) {
//	    lintTask = pocket.WithOpts(Lint, cfg.lint)
//	}
//
//	// In CLI option parsing:
//	taskWithOpts := pocket.WithOpts(task, parsedOpts)
func WithOpts(task *TaskDef, opts any) *TaskDef {
	return &TaskDef{
		name:   task.name,
		usage:  task.usage,
		body:   task.body,
		opts:   opts,
		hidden: task.hidden,
		silent: task.silent,
	}
}

// Clone creates a copy of a task with modifications applied.
// This is useful for creating task variants at runtime, such as
// renaming a task to avoid duplicate names in ManualRun.
//
// Example:
//
//	// Give a task a different name for ManualRun
//	pocket.Clone(golang.Test, pocket.Named("integration-test"))
//
//	// Clone with multiple modifications
//	pocket.Clone(myTask, pocket.Named("new-name"), pocket.AsHidden())
func Clone(task *TaskDef, opts ...TaskOpt) *TaskDef {
	td := &TaskDef{
		name:   task.name,
		usage:  task.usage,
		body:   task.body,
		opts:   task.opts,
		hidden: task.hidden,
		silent: task.silent,
	}
	for _, opt := range opts {
		opt(td)
	}
	return td
}

// Run executes this function with the given context.
// This is useful for testing or programmatic execution.
func (f *TaskDef) Run(ctx context.Context) error {
	return f.run(ctx)
}

// run executes this function with the given context.
// This is called by the framework - users should not call this directly.
func (f *TaskDef) run(ctx context.Context) error {
	ec := getExecContext(ctx)

	// In collect mode, register function and collect nested deps from static tree
	if ec.mode == modeCollect {
		// Check if this would be deduplicated
		deduped := !ec.dedup.shouldRun(runnableKey(f))
		ec.plan.addFunc(f, deduped)
		defer ec.plan.popFunc()

		// Only recurse into Runnable body - do NOT call function bodies
		// This ensures collection is side-effect free and only sees static composition
		if f.body != nil {
			return f.body.run(ctx)
		}
		// Plain functions are wrapped as funcRunnable and not called during collection
		return nil
	}

	// Check skip rules before executing
	if ec.shouldSkipTask(f.name) {
		return nil
	}

	// Execute mode - print task header (skip for hidden or silent tasks)
	if !f.hidden && !f.silent {
		printTaskHeader(ctx, f.name)
	}

	// Inject options into context if present
	if f.opts != nil {
		ctx = withOptions(ctx, f.opts)
	}

	// Execute the Runnable body
	return f.body.run(ctx)
}


// Runnable is the interface for anything that can be executed.
// It uses unexported methods to prevent external implementation,
// ensuring only pocket types (TaskDef, serial, parallel, PathFilter) can satisfy it.
//
// Users create Runnables via:
//   - pocket.Task() for individual functions
//   - pocket.Serial() for sequential execution
//   - pocket.Parallel() for concurrent execution
//   - pocket.RunIn() for path filtering
type Runnable interface {
	run(ctx context.Context) error
}

// toRunnables converts a slice of any to a slice of Runnable.
func toRunnables(items []any) []Runnable {
	result := make([]Runnable, 0, len(items))
	for _, item := range items {
		result = append(result, toRunnable(item))
	}
	return result
}

// toRunnable converts a single item to a Runnable.
func toRunnable(item any) Runnable {
	switch v := item.(type) {
	case Runnable:
		return v
	case func(context.Context) error:
		return &funcRunnable{fn: v}
	default:
		panic("pocket: item must be Runnable or func(context.Context) error")
	}
}

// funcRunnable wraps a plain function as a Runnable.
type funcRunnable struct {
	fn func(context.Context) error
}

func (f *funcRunnable) run(ctx context.Context) error {
	return f.fn(ctx)
}

// commandRunnable executes an external command with static arguments.
type commandRunnable struct {
	name string
	args []string
}

func (c *commandRunnable) run(ctx context.Context) error {
	ec := getExecContext(ctx)
	if ec.mode == modeCollect {
		return nil
	}
	cmd := newCommand(ctx, c.name, c.args...)
	cmd.Stdout = ec.out.Stdout
	cmd.Stderr = ec.out.Stderr
	if ec.path != "" {
		cmd.Dir = FromGitRoot(ec.path)
	} else {
		cmd.Dir = GitRoot()
	}
	return cmd.Run()
}

// Run creates a Runnable that executes an external command.
// The command runs in the current path directory with .pocket/bin in PATH.
//
// Example:
//
//	pocket.Run("go", "fmt", "./...")
//	pocket.Run("golangci-lint", "run", "--fix", "./...")
func Run(name string, args ...string) Runnable {
	return &commandRunnable{name: name, args: args}
}

// doRunnable wraps arbitrary Go code as a Runnable.
type doRunnable struct {
	fn func(context.Context) error
}

func (d *doRunnable) run(ctx context.Context) error {
	ec := getExecContext(ctx)
	if ec.mode == modeCollect {
		return nil
	}
	return d.fn(ctx)
}

// Do creates a Runnable from a function.
// Use this for arbitrary Go code that doesn't fit the Run/RunWith model,
// such as file I/O, API calls, or conditional logic.
//
// Example:
//
//	pocket.Do(func(ctx context.Context) error {
//	    return os.WriteFile("output.txt", data, 0644)
//	})
func Do(fn func(context.Context) error) Runnable {
	return &doRunnable{fn: fn}
}
