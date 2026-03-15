package pk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fredrikaverpil/pocket/internal/scaffold"
	"github.com/fredrikaverpil/pocket/internal/shim"
	"github.com/fredrikaverpil/pocket/pk/repopath"
)

// ErrGitDiffUncommitted is returned when git diff detects uncommitted changes.
var ErrGitDiffUncommitted = errors.New("uncommitted changes detected")

// ErrCommitsInvalid is returned when commit messages fail conventional commit validation.
var ErrCommitsInvalid = errors.New("invalid commit messages")

// builtins is the single source of truth for builtin tasks.
// Used for: lookup, help generation, name conflict checking.
var builtins = []*Task{
	shimsTask,
	planTask,
	gitDiffTask,
	commitsCheckTask,
	selfUpdateTask,
	purgeTask,
}

// isBuiltinName checks if a name is reserved by a builtin.
func isBuiltinName(name string) bool {
	for _, t := range builtins {
		if t.Name == name {
			return true
		}
	}
	return false
}

// shimsTask regenerates shims in all directories.
var shimsTask = &Task{
	Name:       "shims",
	Usage:      "regenerate shims in all directories",
	HideHeader: true,
	Do: func(ctx context.Context) error {
		gitRoot := repopath.GitRoot()
		pocketDir := filepath.Join(gitRoot, ".pocket")

		p := PlanFromContext(ctx)
		if p == nil {
			return fmt.Errorf("plan not found in context")
		}

		// Use shim config from plan (defaults to POSIX only if not configured).
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
	},
}

// planFlags defines flags for the plan task.
type planFlags struct {
	JSON bool `flag:"json" usage:"output as JSON"`
}

// planTask displays the execution plan.
var planTask = &Task{
	Name:       "plan",
	Usage:      "show execution plan without running tasks",
	HideHeader: true,
	Flags:      planFlags{},
	Do: func(ctx context.Context) error {
		p := PlanFromContext(ctx)
		if p == nil {
			return fmt.Errorf("plan not found in context")
		}

		if GetFlags[planFlags](ctx).JSON {
			return printPlanJSON(ctx, p.tree, p)
		}

		// Text output.
		Printf(ctx, "Execution Plan\n")
		Printf(ctx, "==============\n\n")

		// Show module directories where shims will be generated.
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

		// Show composition tree.
		Printf(ctx, "Composition Tree:\n")
		printTree(ctx, p.tree, "", true, "", p)

		Println(ctx)
		Printf(ctx, "Legend: [→] = Serial, [⚡] = Parallel\n")

		return nil
	},
}

// gitDiffTask checks for uncommitted changes.
// Hidden because it's controlled via the -g flag, not direct invocation.
var gitDiffTask = &Task{
	Name:       "git-diff",
	Usage:      "check for uncommitted changes",
	Hidden:     true,
	HideHeader: true,
	Do: func(ctx context.Context) error {
		// Only run if -g flag was passed.
		if !gitDiffEnabledFromContext(ctx) {
			return nil
		}

		Printf(ctx, ":: git-diff\n")
		if err := Exec(ctx, "git", "diff", "--exit-code"); err != nil {
			return ErrGitDiffUncommitted
		}
		return nil
	},
}

// commitsCheckTask validates commit messages against conventional commits.
// Hidden because it's controlled via the -c flag, not direct invocation.
var commitsCheckTask = &Task{
	Name:       "commits-check",
	Usage:      "validate conventional commits after execution",
	Hidden:     true,
	HideHeader: true,
	Do: func(ctx context.Context) error {
		// Only run if -c flag was passed.
		if !commitsCheckEnabledFromContext(ctx) {
			return nil
		}

		Printf(ctx, ":: commits-check\n")

		// Determine commit range: upstream tracking branch or origin/main.
		commitRange, err := resolveCommitRange(ctx)
		if err != nil {
			return fmt.Errorf("resolve commit range: %w", err)
		}
		if commitRange == "" {
			return nil // No commits to validate.
		}

		// Get commit messages using exec.CommandContext directly
		// (pk.Exec buffers output internally and only surfaces it on error).

		cmd := exec.CommandContext(ctx, "git", "log", "--format=%H %s", commitRange)
		cmd.Dir = repopath.GitRoot()
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git log %s: %w\n%s", commitRange, err, out.String())
		}

		// Validate each commit message.
		var invalid []string
		for line := range strings.SplitSeq(out.String(), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			hash, msg, _ := strings.Cut(line, " ")
			shortHash := hash
			if len(shortHash) > 7 {
				shortHash = shortHash[:7]
			}
			if err := ValidateCommitMessage(msg); err != nil {
				invalid = append(invalid, fmt.Sprintf("  %s %q — %s", shortHash, msg, err))
			}
		}

		if len(invalid) > 0 {
			for _, line := range invalid {
				Printf(ctx, "%s\n", line)
			}
			return ErrCommitsInvalid
		}

		return nil
	},
}

// resolveCommitRange determines the git log range for commit validation.
// Returns empty string if there are no commits to validate.
func resolveCommitRange(ctx context.Context) (string, error) {
	gitRoot := repopath.GitRoot()

	// Try upstream tracking branch first.
	//nolint:gosec // Arguments are fixed git commands, not user-supplied.
	cmd := exec.CommandContext(ctx, "git", "log", "--oneline", "@{push}..HEAD")
	cmd.Dir = gitRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err == nil {
		if strings.TrimSpace(out.String()) == "" {
			return "", nil // Zero commits in range.
		}
		return "@{push}..HEAD", nil
	}

	// Fall back to origin's default branch.
	defaultBranch := resolveDefaultBranch(ctx, gitRoot)
	if defaultBranch == "" {
		// Cannot determine default branch (e.g. shallow clone in CI) — silent no-op.
		return "", nil
	}

	ref := "origin/" + defaultBranch + "..HEAD"
	//nolint:gosec // ref is constructed from git output, not user input.
	cmd = exec.CommandContext(ctx, "git", "log", "--oneline", ref)
	cmd.Dir = gitRoot
	out.Reset()
	cmd.Stdout = &out
	cmd.Stderr = &out
	// Ref may not be available (e.g. shallow clone in CI) — treat as no-op.
	if cmd.Run() == nil && strings.TrimSpace(out.String()) != "" {
		return ref, nil
	}
	return "", nil
}

// resolveDefaultBranch returns the default branch name of the origin remote.
// Returns empty string if it cannot be determined.
func resolveDefaultBranch(ctx context.Context, gitRoot string) string {
	// Use git symbolic-ref to find what origin/HEAD points to.
	//nolint:gosec // Arguments are fixed git commands, not user-supplied.
	cmd := exec.CommandContext(ctx, "git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = gitRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	// Output is e.g. "refs/remotes/origin/main\n".
	ref := strings.TrimSpace(out.String())
	return strings.TrimPrefix(ref, "refs/remotes/origin/")
}

// selfUpdateFlags defines flags for the self-update task.
type selfUpdateFlags struct {
	Force bool `flag:"force" usage:"bypass Go proxy cache (slower, but guarantees latest)"`
}

// selfUpdateTask updates Pocket and regenerates scaffolded files.
var selfUpdateTask = &Task{
	Name:  "self-update",
	Usage: "update Pocket and regenerate scaffolded files",
	Flags: selfUpdateFlags{},
	Do: func(ctx context.Context) error {
		gitRoot := repopath.GitRoot()
		pocketDir := filepath.Join(gitRoot, ".pocket")

		// Set working directory to .pocket for all commands.
		ctx = ContextWithPath(ctx, pocketDir)

		// 1. go get latest.
		if GetFlags[selfUpdateFlags](ctx).Force {
			// Bypass proxy cache to guarantee absolute latest.
			if Verbose(ctx) {
				Printf(ctx, "  running: GOPROXY=direct go get github.com/fredrikaverpil/pocket@latest\n")
			}
			ctx := ContextWithEnv(ctx, "GOPROXY=direct")
			if err := Exec(ctx, "go", "get", "github.com/fredrikaverpil/pocket@latest"); err != nil {
				return fmt.Errorf("updating pocket dependency: %w", err)
			}
		} else {
			if Verbose(ctx) {
				Printf(ctx, "  running: go get github.com/fredrikaverpil/pocket@latest\n")
			}
			if err := Exec(ctx, "go", "get", "github.com/fredrikaverpil/pocket@latest"); err != nil {
				return fmt.Errorf("updating pocket dependency: %w", err)
			}
		}

		// 2. go mod tidy.
		if Verbose(ctx) {
			Printf(ctx, "  running: go mod tidy\n")
		}
		if err := Exec(ctx, "go", "mod", "tidy"); err != nil {
			return fmt.Errorf("tidying pocket module: %w", err)
		}

		// 3. Regenerate main.go.
		if Verbose(ctx) {
			Printf(ctx, "  regenerating main.go\n")
		}
		if err := scaffold.RegenerateMain(pocketDir); err != nil {
			return fmt.Errorf("regenerating main.go: %w", err)
		}

		// 4. Regenerate shims.
		return shimsTask.run(ctx)
	},
}

// purgeTask removes .pocket/tools, .pocket/bin, and .pocket/venvs directories.
var purgeTask = &Task{
	Name:  "purge",
	Usage: "remove .pocket/tools, .pocket/bin, and .pocket/venvs",
	Do: func(ctx context.Context) error {
		gitRoot := repopath.GitRoot()
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
	},
}

// --- Plan Helpers ---

// printPlanJSON outputs the plan as JSON.
func printPlanJSON(ctx context.Context, tree Runnable, p *Plan) error {
	output := map[string]any{
		"version":           version(),
		"moduleDirectories": p.moduleDirectories,
		"tree":              buildJSONTree(tree, "", p),
		"tasks":             p.Tasks(), // Use public API - TaskInfo has JSON tags.
	}

	encoder := json.NewEncoder(outputFromContext(ctx).Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// buildJSONTree recursively builds a JSON representation of the composition tree.
// The nameSuffix parameter tracks accumulated name suffixes from WithNameSuffix() wrappers.
// This must match the suffix accumulation logic in plan.go's taskCollector.walk().
func buildJSONTree(r Runnable, nameSuffix string, p *Plan) map[string]any {
	if r == nil {
		return nil
	}

	switch v := r.(type) {
	case *Task:
		effectiveName := v.Name
		if nameSuffix != "" {
			effectiveName = v.Name + ":" + nameSuffix
		}

		paths := []string{"."}
		if info, ok := p.pathMappings[effectiveName]; ok {
			paths = info.resolvedPaths
		}

		manual := false
		if instance := p.taskInstanceByName(effectiveName); instance != nil {
			manual = instance.isManual
		}

		return map[string]any{
			"type":   "task",
			"name":   effectiveName,
			"hidden": v.Hidden,
			"manual": manual,
			"paths":  paths,
		}

	case *serial:
		children := make([]map[string]any, 0, len(v.runnables))
		for _, child := range v.runnables {
			if childJSON := buildJSONTree(child, nameSuffix, p); childJSON != nil {
				children = append(children, childJSON)
			}
		}
		return map[string]any{
			"type":     "serial",
			"children": children,
		}

	case *parallel:
		children := make([]map[string]any, 0, len(v.runnables))
		for _, child := range v.runnables {
			if childJSON := buildJSONTree(child, nameSuffix, p); childJSON != nil {
				children = append(children, childJSON)
			}
		}
		return map[string]any{
			"type":     "parallel",
			"children": children,
		}

	case *pathFilter:
		// Accumulate suffix (matches plan.go logic: "a" + "b" → "a:b").
		childSuffix := nameSuffix
		if v.nameSuffix != "" {
			if nameSuffix != "" {
				childSuffix = nameSuffix + ":" + v.nameSuffix
			} else {
				childSuffix = v.nameSuffix
			}
		}

		node := map[string]any{
			"type":    "pathFilter",
			"include": v.includePaths,
			"exclude": v.excludePaths,
			"inner":   buildJSONTree(v.inner, childSuffix, p),
		}
		return node
	}

	return map[string]any{
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
	p *Plan,
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
		if v.Hidden {
			markers = append(markers, "hidden")
		}

		effectiveName := v.Name
		if nameSuffix != "" {
			effectiveName = v.Name + ":" + nameSuffix
		}

		if instance := p.taskInstanceByName(effectiveName); instance != nil && instance.isManual {
			markers = append(markers, "manual")
		}

		marker := ""
		if len(markers) > 0 {
			marker = " [" + strings.Join(markers, ", ") + "]"
		}

		paths := "[root]"
		if info, ok := p.pathMappings[effectiveName]; ok {
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
			printTree(ctx, child, childPrefix, i == len(v.runnables)-1, nameSuffix, p)
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
			printTree(ctx, child, childPrefix, i == len(v.runnables)-1, nameSuffix, p)
		}

	case *pathFilter:
		// Accumulate suffix (matches plan.go logic: "a" + "b" → "a:b").
		childSuffix := nameSuffix
		if v.nameSuffix != "" {
			if nameSuffix != "" {
				childSuffix = nameSuffix + ":" + v.nameSuffix
			} else {
				childSuffix = v.nameSuffix
			}
		}

		// Only show "With paths" wrapper if there are actual path options.
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
			printTree(ctx, v.inner, childPrefix, true, childSuffix, p)
		} else {
			// No path options - pass through to inner without wrapper.
			printTree(ctx, v.inner, prefix, isLast, childSuffix, p)
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
