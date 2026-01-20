package pk

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/fredrikaverpil/pocket/internal/shim"
)

// Version is the current version of Pocket.
const Version = "2.0.0-dev"

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
		fmt.Printf("pocket %s\n", Version)
		return
	}

	// Build plan
	var plan *plan
	if cfg.Root != nil {
		var err error
		plan, err = newPlan(cfg.Root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building plan: %v\n", err)
			os.Exit(1)
		}
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

		// Execute the task
		ctx := WithVerbose(context.Background(), *verbose)
		if err := executeTask(ctx, task, plan); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Execute the full configuration with pre-built plan
	ctx := WithVerbose(context.Background(), *verbose)
	if err := execute(ctx, *cfg, plan); err != nil {
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

	// Check POK_CONTEXT environment variable (set by shim).
	if pokContext := os.Getenv("POK_CONTEXT"); pokContext != "" {
		// Running from a subdirectory via shim - only run in that path.
		paths = []string{pokContext}
	} else if p != nil {
		// Running from root - use paths from plan.
		if info, ok := p.pathMappings[task.Name()]; ok && len(info.resolvedPaths) > 0 {
			paths = info.resolvedPaths
		} else {
			// No path mapping - run at root.
			paths = []string{"."}
		}
	} else {
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
	if c.Root == nil || p == nil {
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
	return c.Root.run(ctx)
}

// printHelp prints help information including available tasks.
func printHelp(cfg *Config, plan *plan) {
	fmt.Printf("pocket %s\n\n", Version)
	fmt.Println("Usage:")
	fmt.Println("  pok [flags]")
	fmt.Println("  pok <task> [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -h          show help")
	fmt.Println("  -v          verbose mode")
	fmt.Println("  --version   show version")
	fmt.Println()
	fmt.Println("Builtin commands:")
	fmt.Println("  plan [-json]  show execution plan without running tasks")
	fmt.Println()

	// Check if we have tasks
	if plan == nil || len(plan.tasks) == 0 {
		fmt.Println("No tasks configured.")
		return
	}

	// Filter out hidden tasks and sort
	var visibleTasks []*Task
	for _, task := range plan.tasks {
		if !task.IsHidden() {
			visibleTasks = append(visibleTasks, task)
		}
	}

	if len(visibleTasks) == 0 {
		fmt.Println("No visible tasks configured.")
		return
	}

	// Sort by name
	sort.Slice(visibleTasks, func(i, j int) bool {
		return visibleTasks[i].Name() < visibleTasks[j].Name()
	})

	fmt.Println("Available tasks:")
	for _, task := range visibleTasks {
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
		return printPlanJSON(cfg.Root, plan)
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
	printTree(cfg.Root, "", true, plan.pathMappings)

	fmt.Println()
	fmt.Printf("Legend: [→] = Serial, [⚡] = Parallel\n")

	return nil
}

// printPlanJSON outputs the plan as JSON.
func printPlanJSON(root Runnable, plan *plan) error {
	output := map[string]interface{}{
		"version":           Version,
		"moduleDirectories": plan.moduleDirectories,
		"tree":              buildJSONTree(root, plan.pathMappings),
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
		marker := ""
		if v.IsHidden() {
			marker = " [hidden]"
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
