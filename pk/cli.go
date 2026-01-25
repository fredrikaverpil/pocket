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
		case "shims":
			return nil, generateTask.run(ctx)
		case "self-update":
			return nil, updateTask.run(ctx)
		case "purge":
			return nil, cleanTask.run(ctx)
		}
	}

	// Handle specific task execution
	if len(args) > 0 {
		taskName := args[0]
		taskArgs := args[1:]

		// Find task by name
		entry := findTaskByName(plan, taskName)
		if entry == nil {
			return nil, fmt.Errorf("unknown task %q\nRun 'pok -h' to see available tasks", taskName)
		}

		// Check for task-specific help
		if hasHelpFlag(taskArgs) {
			printTaskHelp(ctx, entry.task)
			return nil, nil
		}

		// Parse task flags if present
		if entry.task.Flags() != nil && len(taskArgs) > 0 {
			if err := entry.task.Flags().Parse(taskArgs); err != nil {
				return nil, fmt.Errorf("parsing flags for task %q: %w", taskName, err)
			}
		}

		return executeTask(ctx, entry, plan)
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

// findTaskByName looks up a task entry by name in the Plan.
// The name can be an effective name including suffix (e.g., "py-test:3.9").
// Returns the full taskEntry which includes context values for this task instance.
func findTaskByName(p *Plan, name string) *taskEntry {
	if p == nil {
		return nil
	}
	for i := range p.taskEntries {
		if p.taskEntries[i].name == name {
			return &p.taskEntries[i]
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
	// Wrap the task in a minimal taskEntry for the internal function.
	entry := &taskEntry{task: task, name: task.Name()}
	_, err := executeTask(ctx, entry, p)
	return err
}

// executeTask runs a single task with proper path context and returns the tracker.
func executeTask(ctx context.Context, entry *taskEntry, p *Plan) (*executionTracker, error) {
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
		// Use effective name (entry.name) to look up path mappings.
		if info, ok := p.pathMappings[entry.name]; ok {
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

	// Apply context values from the task entry (e.g., Python version).
	for _, cv := range entry.contextValues {
		ctx = context.WithValue(ctx, cv.key, cv.value)
	}

	// Execute task for each path.
	for _, path := range paths {
		pathCtx := WithPath(ctx, path)
		if err := entry.task.run(pathCtx); err != nil {
			return tracker, fmt.Errorf("task %s in %s: %w", entry.name, path, err)
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
	var regularTasks []taskEntry
	var manualTasks []taskEntry

	if plan != nil && len(plan.taskEntries) > 0 {
		taskScope := os.Getenv("TASK_SCOPE")
		for _, entry := range plan.taskEntries {
			if entry.task.IsHidden() || !plan.taskRunsInPath(entry.name, taskScope) {
				continue
			}
			if entry.task.IsManual() {
				manualTasks = append(manualTasks, entry)
			} else {
				regularTasks = append(regularTasks, entry)
			}
		}
	}

	// Calculate max width for alignment across flags, tasks, and builtins
	allNames := []string{
		// Flags
		"-g, --gitdiff", "-h, --help", "-v, --verbose", "--version",
		// Builtins
		"plan", "shims", "self-update", "purge",
	}
	for _, entry := range regularTasks {
		allNames = append(allNames, entry.name)
	}
	for _, entry := range manualTasks {
		allNames = append(allNames, entry.name)
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
	Printf(ctx, "  %-*s  %s\n", maxWidth, "shims", "regenerate shims in all directories")
	Printf(ctx, "  %-*s  %s\n", maxWidth, "self-update", "update Pocket and regenerate scaffolded files")
	Printf(ctx, "  %-*s  %s\n", maxWidth, "purge", "remove .pocket/tools, .pocket/bin, and .pocket/venvs")
}

// printTaskSection prints a section of tasks with a header.
func printTaskSection(ctx context.Context, header string, entries []taskEntry, width int) {
	if len(entries) == 0 {
		return
	}

	Println(ctx)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	Println(ctx, header)
	for _, entry := range entries {
		if entry.task.Usage() != "" {
			Printf(ctx, "  %-*s  %s\n", width, entry.name, entry.task.Usage())
		} else {
			Printf(ctx, "  %s\n", entry.name)
		}
	}
}
