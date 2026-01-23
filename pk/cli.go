package pk

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
)

// RunMain is the main CLI entry point that handles argument parsing and dispatch.
// It's called from .pocket/main.go.
func RunMain(cfg *Config) {
	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg *Config) error {
	// Parse command-line flags
	fs := flag.NewFlagSet("pok", flag.ExitOnError)
	verbose := fs.Bool("v", false, "verbose mode")
	gitDiff := fs.Bool("g", false, "run git diff check after execution")
	showHelp := fs.Bool("h", false, "show help")
	showVersion := fs.Bool("version", false, "show version")

	// Parse flags
	if err := fs.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	// Set up base context with verbose and output
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx = WithVerbose(ctx, *verbose)
	ctx = withGitDiffEnabled(ctx, *gitDiff)
	ctx = WithOutput(ctx, StdOutput())

	// Handle version flag
	if *showVersion {
		Printf(ctx, "pocket %s\n", version())
		return nil
	}

	// Build Plan
	plan, err := NewPlan(cfg)
	if err != nil {
		return fmt.Errorf("building plan: %w", err)
	}
	ctx = WithPlan(ctx, plan)

	// Handle help flag
	if *showHelp {
		printHelp(ctx, cfg, plan)
		return nil
	}

	// Get remaining arguments (task names)
	args := fs.Args()

	// Handle builtin commands
	if len(args) >= 1 {
		switch args[0] {
		case "plan":
			return handlePlan(ctx, plan, args[1:])
		case "generate":
			// Run generate as a task
			return generateTask.run(ctx)
		case "update":
			// Run update as a task
			return updateTask.run(ctx)
		}
	}

	// Handle specific task execution
	if len(args) > 0 {
		taskName := args[0]
		taskArgs := args[1:]

		// Find task by name
		task := findTaskByName(plan, taskName)
		if task == nil {
			return fmt.Errorf("unknown task %q\nRun 'pok -h' to see available tasks", taskName)
		}

		// Check for task-specific help
		if hasHelpFlag(taskArgs) {
			printTaskHelp(ctx, task)
			return nil
		}

		// Parse task flags if present
		if task.Flags() != nil && len(taskArgs) > 0 {
			if err := task.Flags().Parse(taskArgs); err != nil {
				return fmt.Errorf("parsing flags for task %q: %w", taskName, err)
			}
		}

		return ExecuteTask(ctx, task, plan)
	}

	// Execute the full configuration with pre-built Plan.
	return execute(ctx, *cfg, plan)
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
func ExecuteTask(ctx context.Context, task *Task, p *Plan) error {
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
		if info, ok := p.pathMappings[task.Name()]; ok && len(info.resolvedPaths) > 0 {
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
			return fmt.Errorf("task %s in %s: %w", task.Name(), path, err)
		}
	}

	// Run git diff check after task completes.
	var gitDiffCfg *GitDiffConfig
	if p != nil {
		gitDiffCfg = p.GitDiffConfig()
	}
	return runGitDiff(ctx, gitDiffCfg, tracker)
}

func execute(ctx context.Context, c Config, p *Plan) error {
	if c.Auto == nil || p == nil {
		return nil
	}

	// Generate shims (uses pre-computed ModuleDirectories from Plan)
	if err := generateTask.run(ctx); err != nil {
		return err
	}

	// Execute with Plan and execution tracker in context.
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)
	if err := c.Auto.run(ctx); err != nil {
		return err
	}

	// Run git diff check after all tasks complete (only if -g flag was passed).
	return runGitDiff(ctx, p.GitDiffConfig(), tracker)
}

// printHelp prints help information including available tasks.
func printHelp(ctx context.Context, _ *Config, plan *Plan) {
	Printf(ctx, "pocket %s\n\n", version())
	Println(ctx, "Usage:")
	Println(ctx, "  pok [flags]")
	Println(ctx, "  pok <task> [flags]")
	Println(ctx)
	Println(ctx, "Flags:")
	Println(ctx, "  -g          run git diff check after execution")
	Println(ctx, "  -h          show help")
	Println(ctx, "  -v          verbose mode")
	Println(ctx, "  --version   show version")

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

	printTaskSection(ctx, "Auto tasks:", regularTasks)
	printTaskSection(ctx, "Manual tasks:", manualTasks)

	// Print builtin commands last
	Println(ctx)
	Println(ctx, "Builtin tasks:")
	Println(ctx, "  plan [-json]  show execution plan without running tasks")
	Println(ctx, "  generate      regenerate shims in all directories")
	Println(ctx, "  update        update Pocket and regenerate scaffolded files")
}

// printTaskSection prints a section of tasks with a header.
func printTaskSection(ctx context.Context, header string, tasks []*Task) {
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
			Printf(ctx, "  %-12s  %s\n", task.Name(), task.Usage())
		} else {
			Printf(ctx, "  %s\n", task.Name())
		}
	}
}
