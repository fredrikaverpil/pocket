package pocket

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
)

// Main is the entry point for the CLI.
// It parses flags, handles -h/--help, and runs the specified task(s).
// If no task is specified, defaultTask is run.
//
// pathMappings maps task names to their PathFilter configuration.
// Tasks not in pathMappings are only visible when running from the git root.
//
// autoRunNames is the set of task names from Config.AutoRun (for help display).
func Main(
	tasks []*Task,
	defaultTask *Task,
	pathMappings map[string]*PathFilter,
	autoRunNames map[string]bool,
) {
	os.Exit(run(tasks, defaultTask, pathMappings, autoRunNames))
}

// run parses flags and runs tasks, returning the exit code.
func run(
	tasks []*Task,
	defaultTask *Task,
	pathMappings map[string]*PathFilter,
	autoRunNames map[string]bool,
) int {
	verbose := flag.Bool("v", false, "verbose output")
	help := flag.Bool("h", false, "show help")

	// Detect current working directory relative to git root.
	cwd := detectCwd()

	// Filter tasks based on cwd.
	visibleTasks := filterTasksByCwd(tasks, cwd, pathMappings)

	flag.Usage = func() {
		printHelp(visibleTasks, autoRunNames)
	}
	flag.Parse()

	// Build task map for lookup (visible tasks only).
	taskMap := make(map[string]*Task, len(visibleTasks))
	for _, t := range visibleTasks {
		taskMap[t.Name] = t
	}

	args := flag.Args()

	// Handle help: ./pok -h or ./pok -h taskname
	if *help {
		if len(args) > 0 {
			if t, ok := taskMap[args[0]]; ok {
				printTaskHelp(t)
				return 0
			}
			fmt.Fprintf(os.Stderr, "unknown task: %s\n", args[0])
			return 1
		}
		printHelp(visibleTasks, autoRunNames)
		return 0
	}

	// Parse task name and arguments.
	// Format: pok [flags] <task> [key=value ...]
	var taskToRun *Task
	var taskArgs map[string]string

	if len(args) == 0 {
		if defaultTask != nil {
			taskToRun = defaultTask
		} else {
			fmt.Fprintln(os.Stderr, "no task specified and no default task")
			return 1
		}
	} else {
		// First arg is task name.
		taskName := args[0]
		t, ok := taskMap[taskName]
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown task: %s\n", taskName)
			return 1
		}
		taskToRun = t

		// Parse remaining args as -key=value or -key value pairs.
		var helpRequested bool
		var err error
		taskArgs, helpRequested, err = parseTaskArgs(args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		if helpRequested {
			printTaskHelp(taskToRun)
			return 0
		}
	}

	// Create context with cancellation on interrupt.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create Execution with all state.
	exec := NewExecution(StdOutput(), *verbose, cwd)

	// Set CLI args for the task being run.
	if taskArgs != nil {
		exec.SetTaskArgs(taskToRun.Name, taskArgs)
	}

	// Run the task.
	if err := taskToRun.Run(ctx, exec); err != nil {
		fmt.Fprintf(os.Stderr, "task %s failed: %v\n", taskToRun.Name, err)
		return 1
	}
	return 0
}

// detectCwd returns the current working directory relative to git root.
// Uses POK_CONTEXT environment variable if set (set by the shim script),
// otherwise falls back to detecting from os.Getwd().
// Returns "." if at git root or if detection fails.
func detectCwd() string {
	// Check for POK_CONTEXT environment variable (set by shim).
	if ctx := os.Getenv("POK_CONTEXT"); ctx != "" {
		return ctx
	}

	// Fallback to detecting from actual cwd.
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	root := GitRoot()
	rel, err := filepath.Rel(root, cwd)
	if err != nil {
		return "."
	}
	// Normalize to forward slashes for cross-platform consistency.
	rel = filepath.ToSlash(rel)
	if rel == "" {
		rel = "."
	}
	return rel
}

// filterTasksByCwd returns tasks visible in the given directory.
// - Tasks with path mapping: visible if paths.RunsIn(cwd) returns true
// - Tasks without path mapping: visible only at root (cwd == ".").
func filterTasksByCwd(tasks []*Task, cwd string, pathMappings map[string]*PathFilter) []*Task {
	var result []*Task
	for _, t := range tasks {
		if isTaskVisibleIn(t.Name, cwd, pathMappings) {
			result = append(result, t)
		}
	}
	return result
}

// isTaskVisibleIn returns true if a task should be visible in the given directory.
func isTaskVisibleIn(taskName, cwd string, pathMappings map[string]*PathFilter) bool {
	if paths, ok := pathMappings[taskName]; ok {
		return paths.RunsIn(cwd)
	}
	// Tasks without path mapping are only visible at root.
	return cwd == "."
}

// printHelp prints the help message with available tasks.
// Tasks are grouped into three sections:
// - Tasks: auto-run tasks (from Config.AutoRun)
// - Manual Tasks: explicit-only tasks (from Config.ManualRun)
// - Builtin tasks: core tasks like generate, update, git-diff, clean.
func printHelp(tasks []*Task, autoRunNames map[string]bool) {
	fmt.Println("Usage: pok [flags] <task> [-key=value ...]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -h         show help (use -h <task> or <task> -h for task help)")
	fmt.Println("  -v         verbose output")
	fmt.Println()

	// Separate visible tasks into auto-run, manual, and builtin.
	var autorun, manual, builtin []*Task
	for _, t := range tasks {
		if t.Hidden {
			continue
		}
		switch {
		case t.Builtin:
			builtin = append(builtin, t)
		case autoRunNames[t.Name]:
			autorun = append(autorun, t)
		default:
			manual = append(manual, t)
		}
	}
	sort.Slice(autorun, func(i, j int) bool {
		return autorun[i].Name < autorun[j].Name
	})
	sort.Slice(manual, func(i, j int) bool {
		return manual[i].Name < manual[j].Name
	})
	sort.Slice(builtin, func(i, j int) bool {
		return builtin[i].Name < builtin[j].Name
	})

	if len(autorun) > 0 {
		fmt.Println("Auto-run Tasks:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, t := range autorun {
			fmt.Fprintf(w, "  %s\t%s\n", t.Name, t.Usage)
		}
		w.Flush()
	}

	if len(manual) > 0 {
		if len(autorun) > 0 {
			fmt.Println()
		}
		fmt.Println("Tasks:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, t := range manual {
			fmt.Fprintf(w, "  %s\t%s\n", t.Name, t.Usage)
		}
		w.Flush()
	}

	if len(builtin) > 0 {
		if len(autorun) > 0 || len(manual) > 0 {
			fmt.Println()
		}
		fmt.Println("Builtin tasks:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, t := range builtin {
			fmt.Fprintf(w, "  %s\t%s\n", t.Name, t.Usage)
		}
		w.Flush()
	}

	// Show note if there are no tasks at all.
	if len(autorun) == 0 && len(manual) == 0 && len(builtin) == 0 {
		fmt.Println("No tasks available.")
	}
}

// printTaskHelp prints help for a specific task.
func printTaskHelp(t *Task) {
	fmt.Printf("%s - %s\n", t.Name, t.Usage)

	info, err := inspectArgs(t.Options)
	if err != nil || info == nil || len(info.Fields) == 0 {
		fmt.Println("\nThis task accepts no arguments.")
		return
	}

	fmt.Println("\nArguments:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, field := range info.Fields {
		defaultStr := formatArgDefault(field.Default)
		if field.Usage != "" {
			fmt.Fprintf(w, "  -%s\t%s (default: %s)\n", field.Name, field.Usage, defaultStr)
		} else {
			fmt.Fprintf(w, "  -%s\t(default: %s)\n", field.Name, defaultStr)
		}
	}
	w.Flush()

	fmt.Printf("\nExample:\n  pok %s", t.Name)
	for _, field := range info.Fields {
		fmt.Printf(" -%s=<value>", field.Name)
	}
	fmt.Println()
}

// parseTaskArgs parses task arguments from command line args.
// It supports three formats:
//   - -key=value (explicit value)
//   - -key value (value as next arg, if next arg doesn't start with -)
//   - -key (boolean flag, treated as true)
//
// Returns:
//   - parsed args map
//   - helpRequested: true if -h was found in args
//   - error: if parsing fails
func parseTaskArgs(args []string) (map[string]string, bool, error) {
	taskArgs := make(map[string]string)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-h" {
			return nil, true, nil
		}
		if !strings.HasPrefix(arg, "-") {
			return nil, false, fmt.Errorf("invalid argument %q: expected -key=value or -key value format", arg)
		}
		key := strings.TrimPrefix(arg, "-")
		if k, v, ok := strings.Cut(key, "="); ok {
			// -key=value format
			taskArgs[k] = v
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			// -key value format (next arg is a value, not another flag)
			i++
			taskArgs[key] = args[i]
		} else {
			// -key alone (boolean flag, empty string means true)
			taskArgs[key] = ""
		}
	}
	return taskArgs, false, nil
}
