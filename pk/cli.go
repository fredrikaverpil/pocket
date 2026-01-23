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
	ctx = WithVerbose(ctx, verbose)
	ctx = withGitDiffEnabled(ctx, gitDiff)
	ctx = WithOutput(ctx, StdOutput())

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
	ctx = WithPlan(ctx, plan)

	// Handle help flag
	if showHelp {
		printHelp(ctx, cfg, plan)
		return nil, nil
	}

	// Get remaining arguments (task names)
	args := fs.Args()

	// Handle builtin commands
	if len(args) >= 1 {
		switch args[0] {
		case "plan":
			return nil, handlePlan(ctx, plan, args[1:])
		case "generate":
			return nil, generateTask.run(ctx)
		case "update":
			return nil, updateTask.run(ctx)
		case "clean":
			return nil, cleanTask.run(ctx)
		}
	}

	// Handle specific task execution
	if len(args) > 0 {
		taskName := args[0]
		taskArgs := args[1:]

		// Find task by name
		task := findTaskByName(plan, taskName)
		if task == nil {
			return nil, fmt.Errorf("unknown task %q\nRun 'pok -h' to see available tasks", taskName)
		}

		// Check for task-specific help
		if hasHelpFlag(taskArgs) {
			printTaskHelp(ctx, task)
			return nil, nil
		}

		// Parse task flags if present
		if task.Flags() != nil && len(taskArgs) > 0 {
			if err := task.Flags().Parse(taskArgs); err != nil {
				return nil, fmt.Errorf("parsing flags for task %q: %w", taskName, err)
			}
		}

		return executeTask(ctx, task, plan)
	}

	// Execute the full configuration with pre-built Plan.
	return executeAll(ctx, *cfg, plan)
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

// findTaskByName looks up a task by name in the Plan.
func findTaskByName(p *Plan, name string) *Task {
	if p == nil {
		return nil
	}
	for _, task := range p.tasks {
		if task.Name() == name {
			return task
		}
	}
	return nil
}

// hasHelpFlag checks if the arguments contain a help flag.
func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "-help" {
			return true
		}
	}
	return false
}

// printTaskHelp prints help for a specific task.
func printTaskHelp(ctx context.Context, task *Task) {
	Printf(ctx, "%s - %s\n", task.Name(), task.Usage())
	Println(ctx)
	Printf(ctx, "Usage: pok %s [flags]\n", task.Name())

	if task.Flags() == nil {
		Println(ctx)
		Println(ctx, "This task accepts no flags.")
		return
	}

	Println(ctx)
	Println(ctx, "Flags:")
	task.Flags().SetOutput(OutputFromContext(ctx).Stdout)
	task.Flags().PrintDefaults()
}

// ExecuteTask runs a single task with proper path context.
// This is the public API for external callers.
func ExecuteTask(ctx context.Context, task *Task, p *Plan) error {
	_, err := executeTask(ctx, task, p)
	return err
}

// executeTask runs a single task with proper path context and returns the tracker.
func executeTask(ctx context.Context, task *Task, p *Plan) (*executionTracker, error) {
	// Determine execution paths.
	var paths []string

	// Check TASK_SCOPE environment variable (set by shim).
	// "." means root - use all paths from Plan.
	// Any other value means subdirectory shim - only run in that path.
	taskScope := os.Getenv("TASK_SCOPE")
	switch {
	case taskScope != "" && taskScope != ".":
		// Running from a subdirectory via shim - only run in that path.
		paths = []string{taskScope}
	case p != nil:
		// Running from root - use paths from Plan.
		if info, ok := p.pathMappings[task.Name()]; ok {
			// Task is in pathMappings - use resolved paths (may be empty if excluded).
			paths = info.resolvedPaths
		} else {
			// No path mapping - run at root.
			paths = []string{"."}
		}
	default:
		paths = []string{"."}
	}

	// Set up execution tracker.
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// Execute task for each path.
	for _, path := range paths {
		pathCtx := WithPath(ctx, path)
		if err := task.run(pathCtx); err != nil {
			return tracker, fmt.Errorf("task %s in %s: %w", task.Name(), path, err)
		}
	}

	// Run git diff check after task completes (only if -g flag was passed).
	return tracker, runGitDiff(ctx)
}

func executeAll(ctx context.Context, c Config, p *Plan) (*executionTracker, error) {
	if c.Auto == nil || p == nil {
		return nil, nil
	}

	// Generate shims (uses pre-computed ModuleDirectories from Plan)
	if err := generateTask.run(ctx); err != nil {
		return nil, err
	}

	// Execute with Plan and execution tracker in context.
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)
	if err := c.Auto.run(ctx); err != nil {
		return tracker, err
	}

	// Run git diff check after all tasks complete (only if -g flag was passed).
	return tracker, runGitDiff(ctx)
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
	var regularTasks []*Task
	var manualTasks []*Task

	if plan != nil && len(plan.tasks) > 0 {
		taskScope := os.Getenv("TASK_SCOPE")
		for _, task := range plan.tasks {
			if task.IsHidden() || !plan.taskRunsInPath(task.Name(), taskScope) {
				continue
			}
			if task.IsManual() {
				manualTasks = append(manualTasks, task)
			} else {
				regularTasks = append(regularTasks, task)
			}
		}
	}

	// Calculate max width for alignment across flags, tasks, and builtins
	allNames := []string{
		// Flags
		"-g, --gitdiff", "-h, --help", "-v, --verbose", "--version",
		// Builtins
		"plan", "generate", "update", "clean",
	}
	for _, task := range regularTasks {
		allNames = append(allNames, task.Name())
	}
	for _, task := range manualTasks {
		allNames = append(allNames, task.Name())
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

	// Print builtin commands last
	Println(ctx)
	Println(ctx, "Builtin tasks:")
	Printf(ctx, "  %-*s  %s\n", maxWidth, "plan", "show execution plan without running tasks")
	Printf(ctx, "  %-*s  %s\n", maxWidth, "generate", "regenerate shims in all directories")
	Printf(ctx, "  %-*s  %s\n", maxWidth, "update", "update Pocket and regenerate scaffolded files")
	Printf(ctx, "  %-*s  %s\n", maxWidth, "clean", "remove .pocket/tools and .pocket/bin")
}

// printTaskSection prints a section of tasks with a header.
func printTaskSection(ctx context.Context, header string, tasks []*Task, width int) {
	if len(tasks) == 0 {
		return
	}

	Println(ctx)
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Name() < tasks[j].Name()
	})

	Println(ctx, header)
	for _, task := range tasks {
		if task.Usage() != "" {
			Printf(ctx, "  %-*s  %s\n", width, task.Name(), task.Usage())
		} else {
			Printf(ctx, "  %s\n", task.Name())
		}
	}
}
