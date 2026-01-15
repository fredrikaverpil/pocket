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

// Main is the entry point for the CLI (v2).
// It parses flags, handles -h/--help, and runs the specified function(s).
// If no function is specified, runs all autorun functions.
//
// pathMappings maps function names to their PathFilter configuration.
// Functions not in pathMappings are only visible when running from the git root.
// builtinFuncs are always-available tasks shown under "Built-in tasks" in help.
func Main(
	funcs []*FuncDef,
	allFunc *FuncDef,
	cmds []Cmd,
	pathMappings map[string]*PathFilter,
	autoRunNames map[string]bool,
	builtinFuncs []*FuncDef,
) {
	os.Exit(run(funcs, allFunc, cmds, pathMappings, autoRunNames, builtinFuncs))
}

// run parses flags and runs functions, returning the exit code.
func run(
	funcs []*FuncDef,
	allFunc *FuncDef,
	cmds []Cmd,
	pathMappings map[string]*PathFilter,
	autoRunNames map[string]bool,
	builtinFuncs []*FuncDef,
) int {
	verbose := flag.Bool("v", false, "verbose output")
	help := flag.Bool("h", false, "show help")

	// Detect current working directory relative to git root.
	cwd := detectCwd()

	// Filter functions based on cwd.
	visibleFuncs := filterFuncsByCwd(funcs, cwd, pathMappings)

	flag.Usage = func() {
		printHelp2(visibleFuncs, cmds, autoRunNames, builtinFuncs)
	}
	flag.Parse()

	// Build function map for lookup (visible functions + built-in functions).
	funcMap := make(map[string]*FuncDef, len(visibleFuncs)+len(builtinFuncs))
	for _, f := range visibleFuncs {
		funcMap[f.name] = f
	}
	for _, f := range builtinFuncs {
		funcMap[f.name] = f
	}

	// Build command map for lookup.
	cmdMap := make(map[string]Cmd, len(cmds))
	for _, c := range cmds {
		cmdMap[c.Name] = c
	}

	args := flag.Args()

	// Handle help: ./pok -h or ./pok -h funcname
	if *help {
		if len(args) > 0 {
			if f, ok := funcMap[args[0]]; ok {
				printFuncHelp(f)
				return 0
			}
			if c, ok := cmdMap[args[0]]; ok {
				printCmdHelp(c)
				return 0
			}
			fmt.Fprintf(os.Stderr, "unknown function or command: %s\n", args[0])
			return 1
		}
		printHelp2(visibleFuncs, cmds, autoRunNames, builtinFuncs)
		return 0
	}

	// Create context with cancellation on interrupt.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Determine what to run.
	var funcToRun *FuncDef
	var cmdToRun *Cmd
	var cmdArgs []string

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
		} else if c, ok := cmdMap[name]; ok {
			cmdToRun = &c
			cmdArgs = args[1:]
		} else {
			fmt.Fprintf(os.Stderr, "unknown function or command: %s\n", name)
			return 1
		}
	}

	// Run either a function or a command.
	if cmdToRun != nil {
		if err := cmdToRun.Run(ctx, cmdArgs); err != nil {
			fmt.Fprintf(os.Stderr, "command %s failed: %v\n", cmdToRun.Name, err)
			return 1
		}
		return 0
	}

	// Run the function with the new execution model.
	if err := Run(ctx, funcToRun, StdOutput(), cwd, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "function %s failed: %v\n", funcToRun.name, err)
		return 1
	}
	return 0
}

// filterFuncsByCwd returns functions visible in the given directory.
// - Functions with path mapping: visible if paths.RunsIn(cwd) returns true
// - Functions without path mapping: visible only at root (cwd == ".").
func filterFuncsByCwd(funcs []*FuncDef, cwd string, pathMappings map[string]*PathFilter) []*FuncDef {
	var result []*FuncDef
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

// printHelp2 prints the help message with available functions and commands.
func printHelp2(funcs []*FuncDef, cmds []Cmd, autoRunNames map[string]bool, builtinFuncs []*FuncDef) {
	fmt.Println("Usage: pok [flags] <function> [args...]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -h         show help (use -h <function> for function help)")
	fmt.Println("  -v         verbose output")
	fmt.Println()

	// Separate visible functions into auto-run and manual.
	var autorun, other []*FuncDef
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
		fmt.Println("Functions (auto-run with ./pok):")
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
		fmt.Println("Functions:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, f := range other {
			fmt.Fprintf(w, "  %s\t%s\n", f.name, f.usage)
		}
		w.Flush()
	}

	if len(cmds) > 0 {
		if len(autorun) > 0 || len(other) > 0 {
			fmt.Println()
		}
		fmt.Println("Commands:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, c := range cmds {
			fmt.Fprintf(w, "  %s\t%s\n", c.Name, c.Usage)
		}
		w.Flush()
	}

	// Sort and display built-in tasks.
	if len(builtinFuncs) > 0 {
		if len(autorun) > 0 || len(other) > 0 || len(cmds) > 0 {
			fmt.Println()
		}
		fmt.Println("Functions (builtins):")
		sort.Slice(builtinFuncs, func(i, j int) bool {
			return builtinFuncs[i].name < builtinFuncs[j].name
		})
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, f := range builtinFuncs {
			fmt.Fprintf(w, "  %s\t%s\n", f.name, f.usage)
		}
		w.Flush()
	}

	if len(autorun) == 0 && len(other) == 0 && len(cmds) == 0 && len(builtinFuncs) == 0 {
		fmt.Println("No functions or commands available.")
	}
}

// printFuncHelp prints help for a specific function.
func printFuncHelp(f *FuncDef) {
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

// printCmdHelp prints help for a specific command.
func printCmdHelp(c Cmd) {
	fmt.Printf("%s - %s\n", c.Name, c.Usage)
	fmt.Println("\nThis is a manual command that accepts arbitrary arguments.")
}

// parseTaskArgs parses CLI arguments into a map of key=value pairs.
// Returns (args, wantHelp, error).
func parseTaskArgs(args []string) (map[string]string, bool, error) {
	result := make(map[string]string)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-h" || arg == "--help" {
			return nil, true, nil
		}
		if len(arg) == 0 || arg[0] != '-' {
			return nil, false, fmt.Errorf("expected -key=value or -key value, got %q", arg)
		}
		// Remove leading dashes.
		key := arg[1:]
		if len(key) > 0 && key[0] == '-' {
			key = key[1:]
		}
		// Check for -key=value format.
		if idx := indexOf(key, '='); idx >= 0 {
			result[key[:idx]] = key[idx+1:]
			continue
		}
		// Check if next arg is a value.
		if i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
			result[key] = args[i+1]
			i++
			continue
		}
		// Boolean flag.
		result[key] = ""
	}
	return result, false, nil
}

// indexOf returns the index of the first occurrence of c in s, or -1 if not found.
func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
