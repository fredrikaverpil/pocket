package pk

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"syscall"

	"github.com/fredrikaverpil/pocket/internal/shim"
)

// version returns the current version of Pocket.
// It reads version info embedded by Go 1.18+ during `go build`.
func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	// Check if Pocket is a dependency (user's .pocket/ importing us)
	for _, dep := range info.Deps {
		if dep.Path == "github.com/fredrikaverpil/pocket" {
			// v0.0.0 means replace directive - fall through to VCS check
			if dep.Version != "" && dep.Version != "v0.0.0" {
				return dep.Version
			}
			break
		}
	}

	// Try to get VCS info (works when building in the Pocket repo)
	var revision, dirty string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) >= 7 {
				revision = s.Value[:7]
			} else {
				revision = s.Value
			}
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if revision != "" {
		return "dev-" + revision + dirty
	}

	return "dev"
}

// RunMain is the main CLI entry point that handles argument parsing and dispatch.
// It's called from .pocket/main.go.
func RunMain(cfg *Config) {
	// Parse command-line flags
	fs := flag.NewFlagSet("pok", flag.ExitOnError)
	verbose := fs.Bool("v", false, "verbose mode")
	showHelp := fs.Bool("h", false, "show help")
	showVersion := fs.Bool("version", false, "show version")

	// Parse flags
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Handle version flag
	if *showVersion {
		fmt.Printf("pocket %s\n", version())
		return
	}

	// Build plan
	plan, err := newPlan(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building plan: %v\n", err)
		os.Exit(1)
	}

	// Handle help flag
	if *showHelp {
		printHelp(cfg, plan)
		return
	}

	// Get remaining arguments (task names)
	args := fs.Args()

	// Handle builtin "plan" command (with its own flags)
	if len(args) >= 1 && args[0] == "plan" {
		planFs := flag.NewFlagSet("plan", flag.ExitOnError)
		planJSON := planFs.Bool("json", false, "output as JSON")
		if err := planFs.Parse(args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing plan flags: %v\n", err)
			os.Exit(1)
		}
		if err := printPlan(cfg, plan, *planJSON); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Handle specific task execution
	if len(args) > 0 {
		taskName := args[0]
		taskArgs := args[1:]

		// Find task by name
		task := findTaskByName(plan, taskName)
		if task == nil {
			fmt.Fprintf(os.Stderr, "Error: unknown task %q\n", taskName)
			fmt.Fprintf(os.Stderr, "Run 'pok -h' to see available tasks.\n")
			os.Exit(1)
		}

		// Check for task-specific help
		if hasHelpFlag(taskArgs) {
			printTaskHelp(task)
			return
		}

		// Parse task flags if present
		if task.Flags() != nil && len(taskArgs) > 0 {
			if err := task.Flags().Parse(taskArgs); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing flags for task %q: %v\n", taskName, err)
				os.Exit(1)
			}
		}

		// Execute the task with signal handling.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		ctx = WithVerbose(ctx, *verbose)
		ctx = WithOutput(ctx, StdOutput())
		err := executeTask(ctx, task, plan)
		stop()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Execute the full configuration with pre-built plan and signal handling.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	ctx = WithVerbose(ctx, *verbose)
	ctx = WithOutput(ctx, StdOutput())
	err = execute(ctx, *cfg, plan)
	stop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// findTaskByName looks up a task by name in the plan.
func findTaskByName(p *plan, name string) *Task {
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
func printTaskHelp(task *Task) {
	fmt.Printf("%s - %s\n", task.Name(), task.Usage())
	fmt.Println()
	fmt.Printf("Usage: pok %s [flags]\n", task.Name())

	if task.Flags() == nil {
		fmt.Println()
		fmt.Println("This task accepts no flags.")
		return
	}

	fmt.Println()
	fmt.Println("Flags:")
	task.Flags().PrintDefaults()
}

// executeTask runs a single task with proper path context.
func executeTask(ctx context.Context, task *Task, p *plan) error {
	// Determine execution paths.
	var paths []string

	// Check TASK_SCOPE environment variable (set by shim).
	// "." means root - use all paths from plan.
	// Any other value means subdirectory shim - only run in that path.
	taskScope := os.Getenv("TASK_SCOPE")
	switch {
	case taskScope != "" && taskScope != ".":
		// Running from a subdirectory via shim - only run in that path.
		paths = []string{taskScope}
	case p != nil:
		// Running from root - use paths from plan.
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
	ctx = withExecutionTracker(ctx, newExecutionTracker())

	// Execute task for each path.
	for _, path := range paths {
		pathCtx := WithPath(ctx, path)
		if err := task.run(pathCtx); err != nil {
			return fmt.Errorf("task %s in %s: %w", task.Name(), path, err)
		}
	}

	return nil
}

func execute(ctx context.Context, c Config, p *plan) error {
	if c.Auto == nil || p == nil {
		return nil
	}

	// Generate shims (uses pre-computed ModuleDirectories from plan)
	gitRoot := findGitRoot()
	pocketDir := filepath.Join(gitRoot, ".pocket")
	_, err := shim.GenerateShims(
		ctx,
		gitRoot,
		pocketDir,
		p.moduleDirectories,
		shim.Config{
			Posix:      true,
			Windows:    true,
			PowerShell: true,
		},
	)
	if err != nil {
		return fmt.Errorf("generating shims: %w", err)
	}

	// Execute with plan and execution tracker in context.
	ctx = WithPlan(ctx, p)
	ctx = withExecutionTracker(ctx, newExecutionTracker())
	return c.Auto.run(ctx)
}

// printHelp prints help information including available tasks.
func printHelp(_ *Config, plan *plan) {
	fmt.Printf("pocket %s\n\n", version())
	fmt.Println("Usage:")
	fmt.Println("  pok [flags]")
	fmt.Println("  pok <task> [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -h          show help")
	fmt.Println("  -v          verbose mode")
	fmt.Println("  --version   show version")

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

	printTaskSection("Auto tasks:", regularTasks)
	printTaskSection("Manual tasks:", manualTasks)

	// Print builtin commands last
	fmt.Println()
	fmt.Println("Builtin tasks:")
	fmt.Println("  plan [-json]  show execution plan without running tasks")
}

// printTaskSection prints a section of tasks with a header.
func printTaskSection(header string, tasks []*Task) {
	if len(tasks) == 0 {
		return
	}

	fmt.Println()
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Name() < tasks[j].Name()
	})

	fmt.Println(header)
	for _, task := range tasks {
		if task.Usage() != "" {
			fmt.Printf("  %-12s  %s\n", task.Name(), task.Usage())
		} else {
			fmt.Printf("  %s\n", task.Name())
		}
	}
}

// printPlan builds and displays the execution plan without running tasks.
func printPlan(cfg *Config, plan *plan, asJSON bool) error {
	if plan == nil {
		if asJSON {
			fmt.Println("{}")
		} else {
			fmt.Println("No tasks configured.")
		}
		return nil
	}

	if asJSON {
		return printPlanJSON(cfg.Auto, plan)
	}

	// Text output
	fmt.Printf("Execution Plan\n")
	fmt.Printf("==============\n\n")

	// Show module directories where shims will be generated
	if len(plan.moduleDirectories) > 0 {
		fmt.Printf("Shim Generation:\n")
		for _, dir := range plan.moduleDirectories {
			if dir == "." {
				fmt.Printf("  • root\n")
			} else {
				fmt.Printf("  • %s\n", dir)
			}
		}
		fmt.Println()
	}

	// Show composition tree
	fmt.Printf("Composition Tree:\n")
	printTree(cfg.Auto, "", true, plan.pathMappings)

	fmt.Println()
	fmt.Printf("Legend: [→] = Serial, [⚡] = Parallel\n")

	return nil
}

// printPlanJSON outputs the plan as JSON.
func printPlanJSON(tree Runnable, plan *plan) error {
	output := map[string]any{
		"version":           version(),
		"moduleDirectories": plan.moduleDirectories,
		"tree":              buildJSONTree(tree, plan.pathMappings),
		"tasks":             buildTaskList(plan.tasks, plan.pathMappings),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// buildJSONTree recursively builds a JSON representation of the composition tree.
func buildJSONTree(r Runnable, pathMappings map[string]pathInfo) map[string]interface{} {
	if r == nil {
		return nil
	}

	// Type switch on the runnable
	switch v := r.(type) {
	case *Task:
		paths := []string{"."}
		if pathInfo, ok := pathMappings[v.Name()]; ok && len(pathInfo.resolvedPaths) > 0 {
			paths = pathInfo.resolvedPaths
		}

		return map[string]interface{}{
			"type":   "task",
			"name":   v.Name(),
			"hidden": v.IsHidden(),
			"manual": v.IsManual(),
			"paths":  paths,
		}

	case *serial:
		children := make([]map[string]interface{}, 0, len(v.runnables))
		for _, child := range v.runnables {
			if childJSON := buildJSONTree(child, pathMappings); childJSON != nil {
				children = append(children, childJSON)
			}
		}
		return map[string]interface{}{
			"type":     "serial",
			"children": children,
		}

	case *parallel:
		children := make([]map[string]interface{}, 0, len(v.runnables))
		for _, child := range v.runnables {
			if childJSON := buildJSONTree(child, pathMappings); childJSON != nil {
				children = append(children, childJSON)
			}
		}
		return map[string]interface{}{
			"type":     "parallel",
			"children": children,
		}

	case *pathFilter:
		return map[string]interface{}{
			"type":    "pathFilter",
			"include": v.includePaths,
			"exclude": v.excludePaths,
			"inner":   buildJSONTree(v.inner, pathMappings),
		}
	}

	return map[string]interface{}{
		"type": "unknown",
	}
}

// buildTaskList creates a JSON-friendly task list.
func buildTaskList(tasks []*Task, pathMappings map[string]pathInfo) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(tasks))

	for _, task := range tasks {
		paths := []string{"."}
		if pathInfo, ok := pathMappings[task.Name()]; ok && len(pathInfo.resolvedPaths) > 0 {
			paths = pathInfo.resolvedPaths
		}

		taskJSON := map[string]interface{}{
			"name":   task.Name(),
			"hidden": task.IsHidden(),
			"manual": task.IsManual(),
			"paths":  paths,
		}
		result = append(result, taskJSON)
	}

	return result
}

// printTree recursively prints the composition tree structure.
func printTree(r Runnable, prefix string, isLast bool, pathMappings map[string]pathInfo) {
	if r == nil {
		return
	}

	// Determine the branch characters
	branch := "├── "
	if isLast {
		branch = "└── "
	}

	// Type switch on the runnable
	switch v := r.(type) {
	case *Task:
		// Leaf node - print task name and paths
		var markers []string
		if v.IsHidden() {
			markers = append(markers, "hidden")
		}
		if v.IsManual() {
			markers = append(markers, "manual")
		}
		marker := ""
		if len(markers) > 0 {
			marker = " [" + strings.Join(markers, ", ") + "]"
		}

		paths := "[root]"
		if pathInfo, ok := pathMappings[v.Name()]; ok && len(pathInfo.resolvedPaths) > 0 {
			paths = fmt.Sprintf("%v", pathInfo.resolvedPaths)
		}

		fmt.Printf("%s%s%s%s\n", prefix, branch, v.Name(), marker)

		// Add path info on next line with proper indentation
		continuation := "│   "
		if isLast {
			continuation = "    "
		}
		fmt.Printf("%s%s    paths: %s\n", prefix, continuation, paths)

	case *serial:
		// Serial composition node
		fmt.Printf("%s%s[→] Serial\n", prefix, branch)

		// Extend prefix for children
		childPrefix := prefix
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}

		// Print children
		for i, child := range v.runnables {
			printTree(child, childPrefix, i == len(v.runnables)-1, pathMappings)
		}

	case *parallel:
		// Parallel composition node
		fmt.Printf("%s%s[⚡] Parallel\n", prefix, branch)

		// Extend prefix for children
		childPrefix := prefix
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}

		// Print children
		for i, child := range v.runnables {
			printTree(child, childPrefix, i == len(v.runnables)-1, pathMappings)
		}

	case *pathFilter:
		// Path filter wrapper
		fmt.Printf("%s%s[📁] With paths:\n", prefix, branch)

		childPrefix := prefix
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}

		// Show include/exclude patterns
		if len(v.includePaths) > 0 {
			fmt.Printf("%s    include: %v\n", childPrefix, v.includePaths)
		}
		if len(v.excludePaths) > 0 {
			fmt.Printf("%s    exclude: %v\n", childPrefix, v.excludePaths)
		}

		// Print inner runnable
		printTree(v.inner, childPrefix, true, pathMappings)
	}
}
