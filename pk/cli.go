package pk

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"syscall"

	"github.com/fredrikaverpil/pocket/internal/scaffold"
)

// RunMain is the main CLI entry point that handles argument parsing and dispatch.
// It's called from .pocket/main.go.
func RunMain(cfg *Config) {
	tracker, err := run(cfg)
	if err != nil {
		printFinalStatus(tracker, err)
		os.Exit(1)
	}
	printFinalStatus(tracker, nil)
}

func run(cfg *Config) (*executionTracker, error) {
	// Ensure tools/go.mod exists to prevent go mod tidy from scanning downloaded tools.
	// This must happen before any go commands that might scan the module.
	gitRoot := findGitRoot()
	pocketDir := filepath.Join(gitRoot, ".pocket")
	if err := scaffold.EnsureToolsGomod(pocketDir); err != nil {
		return nil, fmt.Errorf("ensuring tools/go.mod: %w", err)
	}

	// Parse command-line flags
	fs := flag.NewFlagSet("pok", flag.ExitOnError)

	var verbose, gitDiff, showHelp, showVersion bool
	fs.BoolVar(&verbose, "v", false, "verbose mode")
	fs.BoolVar(&verbose, "verbose", false, "verbose mode")
	fs.BoolVar(&gitDiff, "g", false, "run git diff check after execution")
	fs.BoolVar(&gitDiff, "gitdiff", false, "run git diff check after execution")
	fs.BoolVar(&showHelp, "h", false, "show help")
	fs.BoolVar(&showHelp, "help", false, "show help")
	fs.BoolVar(&showVersion, "version", false, "show version")

	// Parse flags
	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, fmt.Errorf("parsing flags: %w", err)
	}

	// Set up base context with verbose and output
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx = ContextWithVerbose(ctx, verbose)
	ctx = ContextWithGitDiffEnabled(ctx, gitDiff)
	ctx = context.WithValue(ctx, outputKey{}, StdOutput())

	// Handle version flag
	if showVersion {
		Printf(ctx, "pocket %s\n", version())
		return nil, nil
	}

	// Build Plan
	plan, err := NewPlan(cfg)
	if err != nil {
		return nil, fmt.Errorf("building plan: %w", err)
	}
	ctx = context.WithValue(ctx, planKey{}, plan)

	// Handle help flag
	if showHelp {
		printHelp(ctx, cfg, plan)
		return nil, nil
	}

	// Get remaining arguments (task names)
	args := fs.Args()

	// Handle task execution (builtins + user tasks)
	if len(args) > 0 {
		taskName := args[0]
		taskArgs := args[1:]

		instance := findTask(plan, taskName)
		if instance == nil {
			return nil, fmt.Errorf("unknown task %q\nRun 'pok -h' to see available tasks", taskName)
		}

		// Parse task flags (handles -h/--help via flag.ErrHelp)
		if err := instance.task.Flags().Parse(taskArgs); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printTaskHelp(ctx, instance.task)
				return nil, nil
			}
			return nil, fmt.Errorf("parsing flags for task %q: %w", taskName, err)
		}

		// Check if this is a builtin task
		if isBuiltinName(instance.task.Name()) {
			// Builtins run directly without path context
			return nil, instance.task.run(ctx)
		}

		return executeTask(ctx, instance)
	}

	// Execute the full configuration with pre-built Plan.
	return executeAll(ctx, *cfg, plan)
}

// findTask looks up a task by name, checking builtins first then user tasks.
// Returns the full taskInstance to preserve the effective name with any suffix.
func findTask(plan *Plan, name string) *taskInstance {
	// Check builtins first
	for _, t := range builtins {
		if t.Name() == name {
			return &taskInstance{task: t, name: t.Name()}
		}
	}
	// Check user tasks - returns instance with effective name preserved
	return findTaskByName(plan, name)
}

// printFinalStatus prints success, warning, or error message with TTY-aware emojis.
// All status messages go to stderr to avoid polluting command output (e.g., JSON from gha-matrix).
func printFinalStatus(tracker *executionTracker, err error) {
	isTTY := isTerminal(os.Stdout)

	var emoji, message string

	switch {
	case errors.Is(err, ErrGitDiffUncommitted):
		emoji, message = "🧹", "Pocket detected uncommitted changes"
	case err != nil:
		emoji, message = "💥", fmt.Sprintf("Error: %v", err)
	case tracker != nil && tracker.warnings():
		emoji, message = "👀", "Pocket completed with warnings"
	case tracker != nil:
		emoji, message = "🚀", "Pocket is done!"
	default:
		return
	}

	if isTTY {
		message = emoji + " " + message
	}

	fmt.Fprintln(os.Stderr, message)
}

// findTaskByName looks up a task instance by name in the Plan.
// The name can be an effective name including suffix (e.g., "py-test:3.9").
// Returns the full taskInstance which includes context values for this task instance.
func findTaskByName(p *Plan, name string) *taskInstance {
	if p == nil {
		return nil
	}
	for i := range p.taskInstances {
		if p.taskInstances[i].name == name {
			return &p.taskInstances[i]
		}
	}
	return nil
}

// printTaskHelp prints help for a specific task.
func printTaskHelp(ctx context.Context, task *Task) {
	Printf(ctx, "%s - %s\n", task.Name(), task.Usage())
	Println(ctx)
	Printf(ctx, "Usage: pok %s [flags]\n", task.Name())

	// Check if the FlagSet has any flags defined.
	hasFlags := false
	task.Flags().VisitAll(func(*flag.Flag) { hasFlags = true })

	if !hasFlags {
		Println(ctx)
		Println(ctx, "This task accepts no flags.")
		return
	}

	Println(ctx)
	Println(ctx, "Flags:")
	task.Flags().SetOutput(outputFromContext(ctx).Stdout)
	task.Flags().PrintDefaults()
}

// ExecuteTask runs a single task by name with proper path context.
// The name can include a suffix (e.g., "py-test:3.9") to select a specific task variant.
// This is the public API for external callers.
func ExecuteTask(ctx context.Context, name string, p *Plan) error {
	instance := findTaskByName(p, name)
	if instance == nil {
		return fmt.Errorf("task %q not found in plan", name)
	}
	_, err := executeTask(ctx, instance)
	return err
}

// executeTask runs a single task with proper path context and returns the tracker.
func executeTask(ctx context.Context, instance *taskInstance) (*executionTracker, error) {
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	if err := instance.execute(ctx); err != nil {
		return tracker, err
	}

	// Run git diff check after task completes (only if -g flag was passed).
	return tracker, gitDiffTask.run(ctx)
}

// execute runs the task in its resolved paths with proper directory management.
// This is the single place where "execute a task in its paths" is defined,
// used by both single-task CLI execution and the public ExecuteTask API.
//
// Named "execute" rather than "run" to avoid satisfying the Runnable interface,
// since taskInstance is not a composable Runnable.
func (inst *taskInstance) execute(ctx context.Context) error {
	// Apply context values from the task instance (e.g., Python version).
	for _, cv := range inst.contextValues {
		ctx = context.WithValue(ctx, cv.key, cv.value)
	}

	// Extract and apply name suffix from instance name (e.g., "py-lint:3.9" -> "3.9").
	// This ensures flag overrides are found when task.run() looks up by effective name.
	baseName := inst.task.Name()
	if len(inst.name) > len(baseName) && inst.name[:len(baseName)] == baseName && inst.name[len(baseName)] == ':' {
		suffix := inst.name[len(baseName)+1:]
		ctx = ContextWithNameSuffix(ctx, suffix)
	}

	// Determine execution paths.
	paths := inst.resolvedPaths
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Check TASK_SCOPE environment variable (set by shim).
	// "." means root - use all paths from the instance.
	// Any other value means subdirectory shim - only run in that path.
	if taskScope := os.Getenv("TASK_SCOPE"); taskScope != "" && taskScope != "." {
		paths = []string{taskScope}
	}

	// Execute task for each path.
	for _, path := range paths {
		pathCtx := ContextWithPath(ctx, path)
		if err := inst.task.run(pathCtx); err != nil {
			return fmt.Errorf("task %s in %s: %w", inst.name, path, err)
		}
	}
	return nil
}

func executeAll(ctx context.Context, c Config, p *Plan) (*executionTracker, error) {
	if c.Auto == nil || p == nil {
		return nil, nil
	}

	// Before: generate shims
	if err := shimsTask.run(ctx); err != nil {
		return nil, err
	}

	// Execute with Plan and execution tracker in context.
	// Auto exec mode causes manual tasks to be skipped.
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)
	ctx = ContextWithAutoExec(ctx)
	if err := c.Auto.run(ctx); err != nil {
		return tracker, err
	}

	// After: git diff check (task checks -g flag internally)
	if err := gitDiffTask.run(ctx); err != nil {
		return tracker, err
	}

	return tracker, nil
}

// printHelp prints help information including available tasks.
func printHelp(ctx context.Context, _ *Config, plan *Plan) {
	Printf(ctx, "pocket %s\n\n", version())
	Println(ctx, "Usage:")
	Println(ctx, "  pok [flags]")
	Println(ctx, "  pok <task> [flags]")

	// Filter tasks by visibility and split into regular vs manual:
	// 1. Not hidden
	// 2. Runs in current path context (from TASK_SCOPE env var)
	var regularTasks []taskInstance
	var manualTasks []taskInstance

	if plan != nil && len(plan.taskInstances) > 0 {
		taskScope := os.Getenv("TASK_SCOPE")
		for _, instance := range plan.taskInstances {
			if instance.task.IsHidden() || !plan.taskRunsInPath(instance.name, taskScope) {
				continue
			}
			if instance.isManual {
				manualTasks = append(manualTasks, instance)
			} else {
				regularTasks = append(regularTasks, instance)
			}
		}
	}

	// Calculate max width for alignment across flags, tasks, and builtins
	allNames := []string{
		// Flags
		"-g, --gitdiff", "-h, --help", "-v, --verbose", "--version",
	}
	// Add visible builtin names
	for _, t := range builtins {
		if !t.IsHidden() {
			allNames = append(allNames, t.Name())
		}
	}
	for _, instance := range regularTasks {
		allNames = append(allNames, instance.name)
	}
	for _, instance := range manualTasks {
		allNames = append(allNames, instance.name)
	}

	maxWidth := 0
	for _, name := range allNames {
		if len(name) > maxWidth {
			maxWidth = len(name)
		}
	}

	// Print flags
	Println(ctx)
	Println(ctx, "Flags:")
	Printf(ctx, "  %-*s  %s\n", maxWidth, "-g, --gitdiff", "run git diff check after execution")
	Printf(ctx, "  %-*s  %s\n", maxWidth, "-h, --help", "show help")
	Printf(ctx, "  %-*s  %s\n", maxWidth, "-v, --verbose", "verbose mode")
	Printf(ctx, "  %-*s  %s\n", maxWidth, "--version", "show version")

	printTaskSection(ctx, "Auto tasks:", regularTasks, maxWidth)
	printTaskSection(ctx, "Manual tasks:", manualTasks, maxWidth)

	// Print builtin commands from builtins slice
	Println(ctx)
	Println(ctx, "Builtin tasks:")
	for _, t := range builtins {
		if !t.IsHidden() {
			Printf(ctx, "  %-*s  %s\n", maxWidth, t.Name(), t.Usage())
		}
	}
}

// printTaskSection prints a section of tasks with a header.
func printTaskSection(ctx context.Context, header string, instances []taskInstance, width int) {
	if len(instances) == 0 {
		return
	}

	Println(ctx)
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].name < instances[j].name
	})

	Println(ctx, header)
	for _, instance := range instances {
		if instance.task.Usage() != "" {
			Printf(ctx, "  %-*s  %s\n", width, instance.name, instance.task.Usage())
		} else {
			Printf(ctx, "  %s\n", instance.name)
		}
	}
}
