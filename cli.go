package pocket

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"syscall"
	"text/tabwriter"
)

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

// cliMain is the entry point for the CLI.
// It parses flags, handles -h/--help, and runs the specified function(s).
// If no function is specified, runs all autorun functions.
//
// pathMappings maps function names to their PathFilter configuration.
// Functions not in pathMappings are only visible when running from the git root.
// builtinFuncs are always-available tasks shown under "Built-in tasks" in help.
func cliMain(
	funcs []*TaskDef,
	allFunc *TaskDef,
	pathMappings map[string]*PathFilter,
	autoRunNames map[string]bool,
	builtinFuncs []*TaskDef,
) {
	os.Exit(cliRun(funcs, allFunc, pathMappings, autoRunNames, builtinFuncs))
}

// cliRun parses flags and runs functions, returning the exit code.
func cliRun(
	funcs []*TaskDef,
	allFunc *TaskDef,
	pathMappings map[string]*PathFilter,
	autoRunNames map[string]bool,
	builtinFuncs []*TaskDef,
) int {
	verbose := flag.Bool("v", false, "verbose output")
	help := flag.Bool("h", false, "show help")

	// Detect current working directory relative to git root.
	cwd := detectCwd()

	// Filter functions based on cwd.
	visibleFuncs := filterFuncsByCwd(funcs, cwd, pathMappings)

	flag.Usage = func() {
		printHelp(visibleFuncs, autoRunNames, builtinFuncs)
	}
	flag.Parse()

	// Build function map for lookup (visible functions + built-in functions).
	funcMap := make(map[string]*TaskDef, len(visibleFuncs)+len(builtinFuncs))
	for _, f := range visibleFuncs {
		funcMap[f.name] = f
	}
	for _, f := range builtinFuncs {
		funcMap[f.name] = f
	}

	args := flag.Args()

	// Handle help: ./pok -h or ./pok -h funcname
	if *help {
		if len(args) > 0 {
			if f, ok := funcMap[args[0]]; ok {
				printFuncHelp(f)
				return 0
			}
			fmt.Fprintf(os.Stderr, "unknown function: %s\n", args[0])
			return 1
		}
		printHelp(visibleFuncs, autoRunNames, builtinFuncs)
		return 0
	}

	// Create context with cancellation on interrupt.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Determine what to run.
	var funcToRun *TaskDef

	if len(args) == 0 {
		// No arguments: run all autorun functions.
		if allFunc != nil {
			funcToRun = allFunc
		} else {
			fmt.Fprintln(os.Stderr, "no function specified and no default function")
			return 1
		}
	} else {
		name := args[0]
		// Check if it's a function.
		if f, ok := funcMap[name]; ok {
			funcToRun = f
			// Parse function-specific arguments.
			if len(args) > 1 && f.opts != nil {
				funcArgs, wantHelp, err := parseTaskArgs(args[1:])
				if err != nil {
					fmt.Fprintf(os.Stderr, "error parsing arguments: %v\n", err)
					return 1
				}
				if wantHelp {
					printFuncHelp(f)
					return 0
				}
				// Parse options and store in function.
				parsedOpts, err := parseOptionsFromCLI(f.opts, funcArgs)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error parsing options: %v\n", err)
					return 1
				}
				if parsedOpts != nil {
					funcToRun = f.With(parsedOpts)
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "unknown function: %s\n", name)
			return 1
		}
	}

	// Run the function.
	if err := runWithContext(ctx, funcToRun, StdOutput(), cwd, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "function %s failed: %v\n", funcToRun.name, err)
		return 1
	}
	return 0
}

// filterFuncsByCwd returns functions visible in the given directory.
// - Functions with path mapping: visible if paths.RunsIn(cwd) returns true
// - Functions without path mapping: visible only at root (cwd == ".").
func filterFuncsByCwd(funcs []*TaskDef, cwd string, pathMappings map[string]*PathFilter) []*TaskDef {
	var result []*TaskDef
	for _, f := range funcs {
		if isFuncVisibleIn(f.name, cwd, pathMappings) {
			result = append(result, f)
		}
	}
	return result
}

// isFuncVisibleIn returns true if a function should be visible in the given directory.
func isFuncVisibleIn(funcName, cwd string, pathMappings map[string]*PathFilter) bool {
	if paths, ok := pathMappings[funcName]; ok {
		return paths.RunsIn(cwd)
	}
	// Functions without path mapping are only visible at root.
	return cwd == "."
}

// printHelp prints the help message with available functions.
func printHelp(funcs []*TaskDef, autoRunNames map[string]bool, builtinFuncs []*TaskDef) {
	fmt.Println("Usage: pok [flags] <task> [args...]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -h         show help (use -h <task> for task help)")
	fmt.Println("  -v         verbose output")
	fmt.Println()

	// Separate visible tasks into auto-run and manual.
	var autorun, other []*TaskDef
	for _, f := range funcs {
		if f.hidden {
			continue
		}
		if autoRunNames[f.name] {
			autorun = append(autorun, f)
		} else {
			other = append(other, f)
		}
	}
	sort.Slice(autorun, func(i, j int) bool {
		return autorun[i].name < autorun[j].name
	})
	sort.Slice(other, func(i, j int) bool {
		return other[i].name < other[j].name
	})

	if len(autorun) > 0 {
		fmt.Println("Tasks (auto-run):")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, f := range autorun {
			fmt.Fprintf(w, "  %s\t%s\n", f.name, f.usage)
		}
		w.Flush()
	}

	if len(other) > 0 {
		if len(autorun) > 0 {
			fmt.Println()
		}
		fmt.Println("Tasks (manual):")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, f := range other {
			fmt.Fprintf(w, "  %s\t%s\n", f.name, f.usage)
		}
		w.Flush()
	}

	// Sort and display built-in tasks.
	if len(builtinFuncs) > 0 {
		if len(autorun) > 0 || len(other) > 0 {
			fmt.Println()
		}
		fmt.Println("Tasks (built-in):")
		sort.Slice(builtinFuncs, func(i, j int) bool {
			return builtinFuncs[i].name < builtinFuncs[j].name
		})
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, f := range builtinFuncs {
			fmt.Fprintf(w, "  %s\t%s\n", f.name, f.usage)
		}
		w.Flush()
	}

	if len(autorun) == 0 && len(other) == 0 && len(builtinFuncs) == 0 {
		fmt.Println("No tasks available.")
	}
}

// printFuncHelp prints help for a specific function.
func printFuncHelp(f *TaskDef) {
	fmt.Printf("%s - %s\n", f.name, f.usage)

	// Check if function has options attached.
	if f.opts == nil {
		fmt.Println("\nThis function accepts no options.")
		return
	}

	info, err := inspectArgs(f.opts)
	if err != nil || info == nil || len(info.Fields) == 0 {
		fmt.Println("\nThis function accepts no options.")
		return
	}

	fmt.Println("\nOptions:")
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
}

// parseTaskArgs and other option parsing functions are in options.go,
// which delegates to internal/cli.
