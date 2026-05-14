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
	"github.com/fredrikaverpil/pocket/pk/internal/ctxkey"
	"github.com/fredrikaverpil/pocket/pk/repopath"
	pkrun "github.com/fredrikaverpil/pocket/pk/run"
)

// RunMain is the CLI entry point for Pocket. It parses arguments, builds
// the execution plan, and dispatches task execution. Call this from your
// .pocket/main.go with the project's [Config].
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
	gitRoot := repopath.GitRoot()
	pocketDir := filepath.Join(gitRoot, ".pocket")
	if err := scaffold.EnsureToolsGomod(pocketDir); err != nil {
		return nil, fmt.Errorf("ensuring tools/go.mod: %w", err)
	}

	// Parse command-line flags
	globalFlags := flag.NewFlagSet("pok", flag.ExitOnError)

	var verbose, serial, gitDiff, commitsCheck, showHelp, showVersion, jsonOut bool
	globalFlags.BoolVar(&verbose, "v", false, "verbose mode")
	globalFlags.BoolVar(&verbose, "verbose", false, "verbose mode")
	globalFlags.BoolVar(&serial, "s", false, "force serial execution (disables parallelism and output buffering)")
	globalFlags.BoolVar(&serial, "serial", false, "force serial execution (disables parallelism and output buffering)")
	globalFlags.BoolVar(&gitDiff, "g", false, "run git diff check after execution")
	globalFlags.BoolVar(&gitDiff, "gitdiff", false, "run git diff check after execution")
	globalFlags.BoolVar(&commitsCheck, "c", false, "validate conventional commits after execution")
	globalFlags.BoolVar(&commitsCheck, "commits", false, "validate conventional commits after execution")
	globalFlags.BoolVar(&showHelp, "h", false, "show help")
	globalFlags.BoolVar(&showHelp, "help", false, "show help")
	globalFlags.BoolVar(&showVersion, "version", false, "show version")
	globalFlags.BoolVar(&jsonOut, "j", false, "emit task plan as JSON instead of executing")
	globalFlags.BoolVar(&jsonOut, "json", false, "emit task plan as JSON instead of executing")

	// Parse flags
	if err := globalFlags.Parse(os.Args[1:]); err != nil {
		return nil, fmt.Errorf("parsing flags: %w", err)
	}

	// Set up base context with verbose and output
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx = context.WithValue(ctx, ctxkey.Verbose{}, verbose)
	ctx = context.WithValue(ctx, ctxkey.Serial{}, serial)
	ctx = context.WithValue(ctx, ctxkey.GitDiff{}, gitDiff)
	ctx = context.WithValue(ctx, ctxkey.CommitsCheck{}, commitsCheck)
	ctx = context.WithValue(ctx, ctxkey.Output{}, pkrun.StdOutput())

	// Handle version flag
	if showVersion {
		pkrun.Printf(ctx, "pocket %s\n", version())
		return nil, nil
	}

	// Build Plan
	plan, err := newPublicPlan(cfg)
	if err != nil {
		return nil, fmt.Errorf("building plan: %w", err)
	}
	ctx = context.WithValue(ctx, ctxkey.Plan{}, plan)

	// Handle help flag
	if showHelp {
		printHelp(ctx, cfg, plan)
		return nil, nil
	}

	// Get remaining arguments (task names)
	remaining := globalFlags.Args()

	// Handle -json: emit instead of executing.
	if jsonOut {
		var taskName string
		if len(remaining) > 0 {
			taskName = remaining[0]
		}
		if err := emitInvocationJSON(ctx, plan, taskName, stdoutFromContext(ctx)); err != nil {
			return nil, err
		}
		return nil, nil
	}

	// Handle task execution (builtins + user tasks)
	if len(remaining) > 0 {
		taskName := remaining[0]
		taskArgs := remaining[1:]

		instance := findTask(plan, taskName)
		if instance == nil {
			return nil, fmt.Errorf("unknown task %q\nRun 'pok -h' to see available tasks", taskName)
		}

		// Parse task flags (handles -h/--help via flag.ErrHelp).
		if instance.task.flagSet != nil {
			if err := instance.task.flagSet.Parse(taskArgs); err != nil {
				if errors.Is(err, flag.ErrHelp) {
					printTaskHelp(ctx, instance.task)
					return nil, nil
				}
				return nil, fmt.Errorf("parsing flags for task %q: %w", taskName, err)
			}
			ctx = context.WithValue(ctx, ctxkey.TaskArgs{}, instance.task.flagSet.Args())
			// Extract only explicitly-set CLI flags (not defaults) for
			// highest-priority override in task.run().
			cliFlags := make(map[string]any)
			instance.task.flagSet.Visit(func(f *flag.Flag) {
				if getter, ok := f.Value.(flag.Getter); ok {
					cliFlags[f.Name] = getter.Get()
				}
			})
			if len(cliFlags) > 0 {
				ctx = context.WithValue(ctx, ctxkey.CLIFlags{}, cliFlags)
			}
		}

		// Check if this is a builtin task.
		if isBuiltinName(instance.task.Name) {
			// Builtins run directly without path context.
			if err := instance.task.run(ctx); err != nil {
				return nil, err
			}
			if instance.task.Name == execTask.Name {
				return nil, nil
			}
			return nil, runPostActions(ctx)
		}

		return executeTask(ctx, instance)
	}

	// Execute the full configuration with pre-built Plan.
	return executeAll(ctx, *cfg, plan)
}

// findTask looks up a task by name, checking builtins first then user tasks.
func findTask(plan *Plan, name string) *taskInstance {
	for _, t := range builtins {
		if t.Name == name {
			if t.flagSet == nil {
				_ = t.buildFlagSet()
			}
			return &taskInstance{task: t, name: t.Name}
		}
	}
	return findTaskByName(plan, name)
}

// printFinalStatus prints success, warning, or error message with TTY-aware emojis.
func printFinalStatus(tracker *executionTracker, err error) {
	isTTY := pkrun.IsTerminal(os.Stdout)

	var emoji, message string

	switch {
	case errors.Is(err, errGitDiffUncommitted):
		emoji, message = "🧹", "Pocket detected uncommitted changes"
	case errors.Is(err, errCommitsInvalid):
		emoji, message = "📝", "Pocket detected invalid commit messages"
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
func findTaskByName(p *Plan, name string) *taskInstance {
	if p == nil {
		return nil
	}
	return p.taskInstanceByName(name)
}

// printTaskHelp prints help for a specific task.
func printTaskHelp(ctx context.Context, task *Task) {
	pkrun.Printf(ctx, "%s - %s\n", task.Name, task.Usage)
	pkrun.Println(ctx)
	pkrun.Printf(ctx, "Usage: pok %s [flags]\n", task.Name)

	if task.flagSet == nil {
		pkrun.Println(ctx)
		pkrun.Println(ctx, "This task accepts no flags.")
		return
	}

	hasFlags := false
	task.flagSet.VisitAll(func(*flag.Flag) { hasFlags = true })

	if !hasFlags {
		pkrun.Println(ctx)
		pkrun.Println(ctx, "This task accepts no flags.")
		return
	}

	pkrun.Println(ctx)
	pkrun.Println(ctx, "Flags:")
	out := pkrun.OutputFromContext(ctx)
	if out == nil {
		out = pkrun.StdOutput()
	}
	task.flagSet.SetOutput(out.Stdout)
	task.flagSet.PrintDefaults()
}

// ExecuteTask runs a single task by name from a pre-built [Plan].
func ExecuteTask(ctx context.Context, name string, p *Plan) error {
	instance := findTaskByName(p, name)
	if instance == nil {
		return fmt.Errorf("task %q not found in plan", name)
	}
	_, err := executeTask(ctx, instance)
	return err
}

// runPostActions runs post-execution checks (git diff, commits validation).
func runPostActions(ctx context.Context) error {
	if err := gitDiffTask.run(ctx); err != nil {
		return err
	}
	return commitsCheckTask.run(ctx)
}

// executeTask runs a single task with proper path context and returns the tracker.
func executeTask(ctx context.Context, instance *taskInstance) (*executionTracker, error) {
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	if err := instance.execute(ctx); err != nil {
		return tracker, err
	}

	return tracker, runPostActions(ctx)
}

// execute runs the task in its resolved paths with proper directory management.
func (inst *taskInstance) execute(ctx context.Context) error {
	// Extract and apply name suffix from instance name (e.g., "py-lint:3.9" -> "3.9").
	baseName := inst.task.Name
	if len(inst.name) > len(baseName) && inst.name[:len(baseName)] == baseName && inst.name[len(baseName)] == ':' {
		suffix := inst.name[len(baseName)+1:]
		ctx = contextWithNameSuffix(ctx, suffix)
	}

	// Determine execution paths.
	paths := inst.resolvedPaths
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Check TASK_SCOPE environment variable (set by shim).
	if taskScope := taskScopeFromEnv(); taskScope != "" {
		paths = []string{taskScope}
	}

	// Execute task for each path.
	for _, path := range paths {
		pathCtx := pkrun.ContextWithPath(ctx, path)
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
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)
	ctx = context.WithValue(ctx, ctxkey.AutoExec{}, true)
	if err := c.Auto.run(ctx); err != nil {
		return tracker, err
	}

	return tracker, runPostActions(ctx)
}

// printHelp prints help information including available tasks.
func printHelp(ctx context.Context, _ *Config, plan *Plan) {
	pkrun.Printf(ctx, "pocket %s\n\n", version())
	pkrun.Println(ctx, "Usage:")
	pkrun.Println(ctx, "  pok [global-flags]")
	pkrun.Println(ctx, "  pok [global-flags] <task> [task-flags]")

	var regularTasks []taskInstance
	var manualTasks []taskInstance

	if plan != nil && len(plan.taskInstances) > 0 {
		taskScope := os.Getenv("TASK_SCOPE")
		for _, instance := range plan.taskInstances {
			if instance.task.Hidden || !plan.taskRunsInPath(instance.name, taskScope) {
				continue
			}
			if instance.isManual {
				manualTasks = append(manualTasks, instance)
			} else {
				regularTasks = append(regularTasks, instance)
			}
		}
	}

	allNames := []string{
		"-c, --commits", "-g, --gitdiff", "-h, --help", "-j, --json",
		"-s, --serial", "-v, --verbose", "--version",
	}
	for _, t := range builtins {
		if !t.Hidden {
			allNames = append(allNames, t.Name)
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

	pkrun.Println(ctx)
	pkrun.Println(ctx, "Global flags:")
	pkrun.Printf(ctx, "  %-*s  %s\n", maxWidth, "-c, --commits", "validate conventional commits after execution")
	pkrun.Printf(ctx, "  %-*s  %s\n", maxWidth, "-g, --gitdiff", "run git diff check after execution")
	pkrun.Printf(ctx, "  %-*s  %s\n", maxWidth, "-h, --help", "show help")
	pkrun.Printf(ctx, "  %-*s  %s\n", maxWidth, "-j, --json", "emit task plan as JSON instead of executing")
	pkrun.Printf(
		ctx,
		"  %-*s  %s\n",
		maxWidth,
		"-s, --serial",
		"force serial execution (disables parallelism and output buffering)",
	)
	pkrun.Printf(ctx, "  %-*s  %s\n", maxWidth, "-v, --verbose", "verbose mode")
	pkrun.Printf(ctx, "  %-*s  %s\n", maxWidth, "--version", "show version")

	printTaskSection(ctx, "Auto tasks:", regularTasks, maxWidth)
	printTaskSection(ctx, "Manual tasks:", manualTasks, maxWidth)

	pkrun.Println(ctx)
	pkrun.Println(ctx, "Builtin tasks:")
	for _, t := range builtins {
		if !t.Hidden {
			pkrun.Printf(ctx, "  %-*s  %s\n", maxWidth, t.Name, t.Usage)
		}
	}

	pkrun.Println(ctx)
	pkrun.Println(ctx, "Run 'pok <task> -h' for task-specific flags.")
}

// printTaskSection prints a section of tasks with a header.
func printTaskSection(ctx context.Context, header string, instances []taskInstance, width int) {
	if len(instances) == 0 {
		return
	}

	pkrun.Println(ctx)
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].name < instances[j].name
	})

	pkrun.Println(ctx, header)
	for _, instance := range instances {
		if instance.task.Usage != "" {
			pkrun.Printf(ctx, "  %-*s  %s\n", width, instance.name, instance.task.Usage)
		} else {
			pkrun.Printf(ctx, "  %s\n", instance.name)
		}
	}
}
