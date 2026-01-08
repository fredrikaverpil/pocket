package pocket

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
)

// Main is the entry point for the CLI.
// It parses flags, handles -h/--help, and runs the specified task(s).
// If no task is specified, defaultTask is run.
func Main(tasks []*Task, defaultTask *Task) {
	os.Exit(run(tasks, defaultTask))
}

// run parses flags and runs tasks, returning the exit code.
func run(tasks []*Task, defaultTask *Task) int {
	verbose := flag.Bool("v", false, "verbose output")
	help := flag.Bool("h", false, "show help")
	flag.Usage = func() {
		printHelp(tasks, defaultTask)
	}
	flag.Parse()

	// Build task map for lookup.
	taskMap := make(map[string]*Task, len(tasks))
	for _, t := range tasks {
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
		printHelp(tasks, defaultTask)
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

		// Remaining args are key=value pairs.
		taskArgs = make(map[string]string)
		for _, arg := range args[1:] {
			key, value, ok := strings.Cut(arg, "=")
			if !ok {
				fmt.Fprintf(os.Stderr, "invalid argument %q: expected key=value format\n", arg)
				return 1
			}
			taskArgs[key] = value
		}
	}

	// Set task arguments.
	taskToRun.SetArgs(taskArgs)

	// Create context with cancellation on interrupt.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Set verbose mode in context.
	ctx = WithVerbose(ctx, *verbose)

	// Run the task.
	if err := Run(ctx, taskToRun); err != nil {
		fmt.Fprintf(os.Stderr, "task %s failed: %v\n", taskToRun.Name, err)
		return 1
	}
	return 0
}

// printHelp prints the help message with available tasks.
func printHelp(tasks []*Task, defaultTask *Task) {
	fmt.Println("Usage: pok [flags] <task> [args...]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -h         show help (use -h <task> for task help)")
	fmt.Println("  -v         verbose output")
	fmt.Println()

	// Separate visible tasks into regular and builtin.
	var regular, builtin []*Task
	for _, t := range tasks {
		if t.Hidden {
			continue
		}
		if t.Builtin {
			builtin = append(builtin, t)
		} else {
			regular = append(regular, t)
		}
	}
	sort.Slice(regular, func(i, j int) bool {
		return regular[i].Name < regular[j].Name
	})
	sort.Slice(builtin, func(i, j int) bool {
		return builtin[i].Name < builtin[j].Name
	})

	fmt.Println("Tasks:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, t := range regular {
		defaultMark := ""
		if defaultTask != nil && t.Name == defaultTask.Name {
			defaultMark = " (default)"
		}
		fmt.Fprintf(w, "  %s\t%s%s\n", t.Name, t.Usage, defaultMark)
	}
	w.Flush()

	if len(builtin) > 0 {
		fmt.Println()
		fmt.Println("Builtin tasks:")
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, t := range builtin {
			fmt.Fprintf(w, "  %s\t%s\n", t.Name, t.Usage)
		}
		w.Flush()
	}
}

// printTaskHelp prints help for a specific task.
func printTaskHelp(t *Task) {
	fmt.Printf("%s - %s\n", t.Name, t.Usage)

	if len(t.Args) == 0 {
		fmt.Println("\nThis task accepts no arguments.")
		return
	}

	fmt.Println("\nArguments:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, arg := range t.Args {
		if arg.Default != "" {
			fmt.Fprintf(w, "  %s\t%s (default: %q)\n", arg.Name, arg.Usage, arg.Default)
		} else {
			fmt.Fprintf(w, "  %s\t%s\n", arg.Name, arg.Usage)
		}
	}
	w.Flush()

	fmt.Printf("\nExample:\n  pok %s", t.Name)
	for _, arg := range t.Args {
		fmt.Printf(" %s=<value>", arg.Name)
	}
	fmt.Println()
}
