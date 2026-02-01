package pk

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fredrikaverpil/pocket/internal/scaffold"
	"github.com/fredrikaverpil/pocket/internal/shim"
)

// ErrGitDiffUncommitted is returned when git diff detects uncommitted changes.
var ErrGitDiffUncommitted = errors.New("uncommitted changes detected")

// builtins is the single source of truth for builtin tasks.
// Used for: lookup, help generation, name conflict checking.
var builtins = []*Task{
	shimsTask,
	planTask,
	gitDiffTask,
	selfUpdateTask,
	purgeTask,
}

// isBuiltinName checks if a name is reserved by a builtin.
func isBuiltinName(name string) bool {
	for _, t := range builtins {
		if t.Name() == name {
			return true
		}
	}
	return false
}

// shimsTask regenerates shims in all directories.
var shimsTask = NewTask("shims", "regenerate shims in all directories", nil, Do(func(ctx context.Context) error {
	gitRoot := findGitRoot()
	pocketDir := filepath.Join(gitRoot, ".pocket")

	p := PlanFromContext(ctx)
	if p == nil {
		return fmt.Errorf("plan not found in context")
	}

	// Use shim config from plan (defaults to POSIX only if not configured)
	cfg := p.ShimConfig()
	shims, err := shim.GenerateShims(
		ctx,
		gitRoot,
		pocketDir,
		p.moduleDirectories,
		shim.Config{
			Posix:      cfg.Posix,
			Windows:    cfg.Windows,
			PowerShell: cfg.PowerShell,
		},
	)
	if err != nil {
		return fmt.Errorf("generating shims: %w", err)
	}

	if Verbose(ctx) {
		for _, s := range shims {
			Printf(ctx, "  generated: %s\n", s)
		}
	}

	return nil
})).HideHeader()

// plan flags.
var (
	planFlags = flag.NewFlagSet("plan", flag.ContinueOnError)
	planJSON  = planFlags.Bool("json", false, "output as JSON")
)

// planTask displays the execution plan.
var planTask = NewTask(
	"plan",
	"show execution plan without running tasks",
	planFlags,
	Do(func(ctx context.Context) error {
		p := PlanFromContext(ctx)
		if p == nil {
			return fmt.Errorf("plan not found in context")
		}

		if *planJSON {
			return printPlanJSON(ctx, p.tree, p)
		}

		// Text output
		Printf(ctx, "Execution Plan\n")
		Printf(ctx, "==============\n\n")

		// Show module directories where shims will be generated
		if len(p.moduleDirectories) > 0 {
			Printf(ctx, "Shim Generation:\n")
			for _, dir := range p.moduleDirectories {
				if dir == "." {
					Printf(ctx, "  • root\n")
				} else {
					Printf(ctx, "  • %s\n", dir)
				}
			}
			Println(ctx)
		}

		// Show composition tree
		Printf(ctx, "Composition Tree:\n")
		printTree(ctx, p.tree, "", true, "", p.pathMappings)

		Println(ctx)
		Printf(ctx, "Legend: [→] = Serial, [⚡] = Parallel\n")

		return nil
	}),
).HideHeader()

// gitDiffTask checks for uncommitted changes.
// Hidden because it's controlled via the -g flag, not direct invocation.
var gitDiffTask = NewTask("git-diff", "check for uncommitted changes", nil, Do(func(ctx context.Context) error {
	// Only run if -g flag was passed
	if !gitDiffEnabledFromContext(ctx) {
		return nil
	}

	Printf(ctx, ":: git-diff\n")
	if err := Exec(ctx, "git", "diff", "--exit-code"); err != nil {
		return ErrGitDiffUncommitted
	}
	return nil
})).Hidden().HideHeader()

// self-update flags.
var (
	selfUpdateFlags = flag.NewFlagSet("self-update", flag.ContinueOnError)
	selfUpdateForce = selfUpdateFlags.Bool("force", false, "bypass Go proxy cache (slower, but guarantees latest)")
)

// selfUpdateTask updates Pocket and regenerates scaffolded files.
var selfUpdateTask = NewTask(
	"self-update",
	"update Pocket and regenerate scaffolded files",
	selfUpdateFlags,
	Do(func(ctx context.Context) error {
		gitRoot := findGitRoot()
		pocketDir := filepath.Join(gitRoot, ".pocket")

		// 1. go get latest
		cmd := exec.CommandContext(ctx, "go", "get", "github.com/fredrikaverpil/pocket@latest")
		cmd.Dir = pocketDir
		if *selfUpdateForce {
			// Bypass proxy cache to guarantee absolute latest
			cmd.Env = append(cmd.Environ(), "GOPROXY=direct")
			if Verbose(ctx) {
				Printf(ctx, "  running: GOPROXY=direct go get github.com/fredrikaverpil/pocket@latest\n")
			}
		} else if Verbose(ctx) {
			Printf(ctx, "  running: go get github.com/fredrikaverpil/pocket@latest\n")
		}
		out := OutputFromContext(ctx)
		cmd.Stdout = out.Stdout
		cmd.Stderr = out.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("updating pocket dependency: %w", err)
		}

		// 2. go mod tidy
		if Verbose(ctx) {
			Printf(ctx, "  running: go mod tidy\n")
		}
		cmd = exec.CommandContext(ctx, "go", "mod", "tidy")
		cmd.Dir = pocketDir
		cmd.Stdout = out.Stdout
		cmd.Stderr = out.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("tidying pocket module: %w", err)
		}

		// 3. Regenerate main.go
		if Verbose(ctx) {
			Printf(ctx, "  regenerating main.go\n")
		}
		if err := scaffold.RegenerateMain(pocketDir); err != nil {
			return fmt.Errorf("regenerating main.go: %w", err)
		}

		// 4. Regenerate shims
		return shimsTask.run(ctx)
	}),
)

// purgeTask removes .pocket/tools, .pocket/bin, and .pocket/venvs directories.
var purgeTask = NewTask(
	"purge",
	"remove .pocket/tools, .pocket/bin, and .pocket/venvs",
	nil,
	Do(func(ctx context.Context) error {
		gitRoot := findGitRoot()
		pocketDir := filepath.Join(gitRoot, ".pocket")

		dirsToRemove := []string{
			filepath.Join(pocketDir, "tools"),
			filepath.Join(pocketDir, "bin"),
			filepath.Join(pocketDir, "venvs"),
		}

		for _, dir := range dirsToRemove {
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("removing %s: %w", dir, err)
			}
			if Verbose(ctx) {
				Printf(ctx, "  removed: %s\n", dir)
			}
		}

		return nil
	}),
)

// --- Plan Helpers ---

// printPlanJSON outputs the plan as JSON.
func printPlanJSON(ctx context.Context, tree Runnable, p *Plan) error {
	output := map[string]any{
		"version":           version(),
		"moduleDirectories": p.moduleDirectories,
		"tree":              buildJSONTree(tree, "", p.pathMappings),
		"tasks":             p.Tasks(), // Use public API - TaskInfo has JSON tags
	}

	encoder := json.NewEncoder(OutputFromContext(ctx).Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// buildJSONTree recursively builds a JSON representation of the composition tree.
// The nameSuffix parameter tracks accumulated name suffixes from WithNameSuffix() wrappers.
// This must match the suffix accumulation logic in plan.go's taskCollector.walk().
func buildJSONTree(r Runnable, nameSuffix string, pathMappings map[string]pathInfo) map[string]interface{} {
	if r == nil {
		return nil
	}

	switch v := r.(type) {
	case *Task:
		effectiveName := v.Name()
		if nameSuffix != "" {
			effectiveName = v.Name() + ":" + nameSuffix
		}

		paths := []string{"."}
		if info, ok := pathMappings[effectiveName]; ok {
			paths = info.resolvedPaths
		}

		return map[string]interface{}{
			"type":   "task",
			"name":   effectiveName,
			"hidden": v.IsHidden(),
			"manual": v.IsManual(),
			"paths":  paths,
		}

	case *serial:
		children := make([]map[string]interface{}, 0, len(v.runnables))
		for _, child := range v.runnables {
			if childJSON := buildJSONTree(child, nameSuffix, pathMappings); childJSON != nil {
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
			if childJSON := buildJSONTree(child, nameSuffix, pathMappings); childJSON != nil {
				children = append(children, childJSON)
			}
		}
		return map[string]interface{}{
			"type":     "parallel",
			"children": children,
		}

	case *pathFilter:
		// Accumulate suffix (matches plan.go logic: "a" + "b" → "a:b")
		childSuffix := nameSuffix
		if v.nameSuffix != "" {
			if nameSuffix != "" {
				childSuffix = nameSuffix + ":" + v.nameSuffix
			} else {
				childSuffix = v.nameSuffix
			}
		}

		node := map[string]interface{}{
			"type":    "pathFilter",
			"include": v.includePaths,
			"exclude": v.excludePaths,
			"inner":   buildJSONTree(v.inner, childSuffix, pathMappings),
		}
		return node
	}

	return map[string]interface{}{
		"type": "unknown",
	}
}

// printTree recursively prints the composition tree structure.
// The nameSuffix parameter tracks accumulated name suffixes from WithNameSuffix() wrappers.
// This must match the suffix accumulation logic in plan.go's taskCollector.walk().
func printTree(
	ctx context.Context,
	r Runnable,
	prefix string,
	isLast bool,
	nameSuffix string,
	pathMappings map[string]pathInfo,
) {
	if r == nil {
		return
	}

	branch := "├── "
	if isLast {
		branch = "└── "
	}

	switch v := r.(type) {
	case *Task:
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

		effectiveName := v.Name()
		if nameSuffix != "" {
			effectiveName = v.Name() + ":" + nameSuffix
		}

		paths := "[root]"
		if info, ok := pathMappings[effectiveName]; ok {
			if len(info.resolvedPaths) > 0 {
				paths = formatPaths(info.resolvedPaths)
			} else {
				paths = "[skipped]"
			}
		}

		Printf(ctx, "%s%s%s%s\n", prefix, branch, effectiveName, marker)

		continuation := "│   "
		if isLast {
			continuation = "    "
		}
		Printf(ctx, "%s%s    paths: %s\n", prefix, continuation, paths)

	case *serial:
		Printf(ctx, "%s%s[→] Serial\n", prefix, branch)
		childPrefix := prefix
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
		for i, child := range v.runnables {
			printTree(ctx, child, childPrefix, i == len(v.runnables)-1, nameSuffix, pathMappings)
		}

	case *parallel:
		Printf(ctx, "%s%s[⚡] Parallel\n", prefix, branch)
		childPrefix := prefix
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
		for i, child := range v.runnables {
			printTree(ctx, child, childPrefix, i == len(v.runnables)-1, nameSuffix, pathMappings)
		}

	case *pathFilter:
		// Accumulate suffix (matches plan.go logic: "a" + "b" → "a:b")
		childSuffix := nameSuffix
		if v.nameSuffix != "" {
			if nameSuffix != "" {
				childSuffix = nameSuffix + ":" + v.nameSuffix
			} else {
				childSuffix = v.nameSuffix
			}
		}

		// Only show "With paths" wrapper if there are actual path options
		hasPathOptions := len(v.includePaths) > 0 || len(v.excludePaths) > 0 ||
			v.detectFunc != nil
		if hasPathOptions {
			Printf(ctx, "%s%s[📁] With paths:\n", prefix, branch)
			childPrefix := prefix
			if isLast {
				childPrefix += "    "
			} else {
				childPrefix += "│   "
			}
			if len(v.includePaths) > 0 {
				Printf(ctx, "%s    include: %v\n", childPrefix, v.includePaths)
			}
			if len(v.excludePaths) > 0 {
				Printf(ctx, "%s    exclude: %v\n", childPrefix, v.excludePaths)
			}
			printTree(ctx, v.inner, childPrefix, true, childSuffix, pathMappings)
		} else {
			// No path options - pass through to inner without wrapper
			printTree(ctx, v.inner, prefix, isLast, childSuffix, pathMappings)
		}
	}
}

// formatPaths formats a path list for display.
// Shows full list if <= 3 paths, otherwise shows count.
func formatPaths(paths []string) string {
	if len(paths) == 0 {
		return "[root]"
	}
	if len(paths) == 1 && paths[0] == "." {
		return "[root]"
	}
	if len(paths) <= 3 {
		return fmt.Sprintf("%v", paths)
	}
	return fmt.Sprintf("%d directories", len(paths))
}
