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
	"github.com/fredrikaverpil/pocket/pk/conventionalcommits"
	"github.com/fredrikaverpil/pocket/pk/repopath"
	pkrun "github.com/fredrikaverpil/pocket/pk/run"
)

// errGitDiffUncommitted is returned when git diff detects uncommitted changes.
var errGitDiffUncommitted = errors.New("uncommitted changes detected")

// errCommitsInvalid is returned when commit messages fail conventional commit validation.
var errCommitsInvalid = errors.New("invalid commit messages")

// builtins is the single source of truth for builtin tasks.
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

		p := planFromContext(ctx)
		if p == nil {
			return fmt.Errorf("plan not found in context")
		}

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

		if pkrun.Verbose(ctx) {
			for _, s := range shims {
				pkrun.Printf(ctx, "  generated: %s\n", s)
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
		p := planFromContext(ctx)
		if p == nil {
			return fmt.Errorf("plan not found in context")
		}

		if pkrun.GetFlags[planFlags](ctx).JSON {
			return printPlanJSON(ctx, p.tree, p)
		}

		// Text output.
		pkrun.Printf(ctx, "Execution Plan\n")
		pkrun.Printf(ctx, "==============\n\n")

		if len(p.moduleDirectories) > 0 {
			pkrun.Printf(ctx, "Shim Generation:\n")
			for _, dir := range p.moduleDirectories {
				if dir == "." {
					pkrun.Printf(ctx, "  • root\n")
				} else {
					pkrun.Printf(ctx, "  • %s\n", dir)
				}
			}
			pkrun.Println(ctx)
		}

		pkrun.Printf(ctx, "Composition Tree:\n")
		printTree(ctx, p.tree, "", true, "", p)

		pkrun.Println(ctx)
		pkrun.Printf(ctx, "Legend: [→] = Serial, [⚡] = Parallel\n")

		return nil
	},
}

// gitDiffTask checks for uncommitted changes.
var gitDiffTask = &Task{
	Name:       "git-diff",
	Usage:      "check for uncommitted changes",
	Hidden:     true,
	HideHeader: true,
	Do: func(ctx context.Context) error {
		if !gitDiffEnabled(ctx) {
			return nil
		}

		pkrun.Printf(ctx, ":: git-diff\n")
		if err := pkrun.Exec(ctx, "git", "diff", "--exit-code"); err != nil {
			return errGitDiffUncommitted
		}
		return nil
	},
}

// commitsCheckTask validates commit messages against conventional commits.
var commitsCheckTask = &Task{
	Name:       "commits-check",
	Usage:      "validate conventional commits after execution",
	Hidden:     true,
	HideHeader: true,
	Do: func(ctx context.Context) error {
		if !commitsCheckEnabled(ctx) {
			return nil
		}

		pkrun.Printf(ctx, ":: commits-check\n")

		commitRange, err := resolveCommitRange(ctx)
		if err != nil {
			return fmt.Errorf("resolve commit range: %w", err)
		}
		if commitRange == "" {
			return nil
		}

		cmd := exec.CommandContext(ctx, "git", "log", "--format=%H %s", commitRange)
		cmd.Dir = repopath.GitRoot()
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git log %s: %w\n%s", commitRange, err, out.String())
		}

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
			if err := conventionalcommits.ValidateMessage(msg); err != nil {
				invalid = append(invalid, fmt.Sprintf("  %s %q — %s", shortHash, msg, err))
			}
		}

		if len(invalid) > 0 {
			for _, line := range invalid {
				pkrun.Printf(ctx, "%s\n", line)
			}
			return errCommitsInvalid
		}

		return nil
	},
}

// resolveCommitRange determines the git log range for commit validation.
func resolveCommitRange(ctx context.Context) (string, error) {
	gitRoot := repopath.GitRoot()

	//nolint:gosec // Arguments are fixed git commands, not user-supplied.
	cmd := exec.CommandContext(ctx, "git", "log", "--oneline", "@{push}..HEAD")
	cmd.Dir = gitRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err == nil {
		if strings.TrimSpace(out.String()) == "" {
			return "", nil
		}
		return "@{push}..HEAD", nil
	}

	defaultBranch := resolveDefaultBranch(ctx, gitRoot)
	if defaultBranch == "" {
		return "", nil
	}

	ref := "origin/" + defaultBranch + "..HEAD"
	//nolint:gosec // ref is constructed from git output, not user input.
	cmd = exec.CommandContext(ctx, "git", "log", "--oneline", ref)
	cmd.Dir = gitRoot
	out.Reset()
	cmd.Stdout = &out
	cmd.Stderr = &out
	if cmd.Run() == nil && strings.TrimSpace(out.String()) != "" {
		return ref, nil
	}
	return "", nil
}

// resolveDefaultBranch returns the default branch name of the origin remote.
func resolveDefaultBranch(ctx context.Context, gitRoot string) string {
	//nolint:gosec // Arguments are fixed git commands, not user-supplied.
	cmd := exec.CommandContext(ctx, "git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = gitRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
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

		ctx = pkrun.ContextWithPath(ctx, pocketDir)

		if pkrun.GetFlags[selfUpdateFlags](ctx).Force {
			if pkrun.Verbose(ctx) {
				pkrun.Printf(ctx, "  running: GOPROXY=direct go get github.com/fredrikaverpil/pocket@latest\n")
			}
			ctx := pkrun.ContextWithEnv(ctx, "GOPROXY=direct")
			if err := pkrun.Exec(ctx, "go", "get", "github.com/fredrikaverpil/pocket@latest"); err != nil {
				return fmt.Errorf("updating pocket dependency: %w", err)
			}
		} else {
			if pkrun.Verbose(ctx) {
				pkrun.Printf(ctx, "  running: go get github.com/fredrikaverpil/pocket@latest\n")
			}
			if err := pkrun.Exec(ctx, "go", "get", "github.com/fredrikaverpil/pocket@latest"); err != nil {
				return fmt.Errorf("updating pocket dependency: %w", err)
			}
		}

		if pkrun.Verbose(ctx) {
			pkrun.Printf(ctx, "  running: go mod tidy\n")
		}
		if err := pkrun.Exec(ctx, "go", "mod", "tidy"); err != nil {
			return fmt.Errorf("tidying pocket module: %w", err)
		}

		if pkrun.Verbose(ctx) {
			pkrun.Printf(ctx, "  regenerating main.go\n")
		}
		if err := scaffold.RegenerateMain(pocketDir); err != nil {
			return fmt.Errorf("regenerating main.go: %w", err)
		}

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
			if pkrun.Verbose(ctx) {
				pkrun.Printf(ctx, "  removed: %s\n", dir)
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
		"tasks":             p.Tasks(),
	}

	out := pkrun.OutputFromContext(ctx)
	if out == nil {
		out = pkrun.StdOutput()
	}
	encoder := json.NewEncoder(out.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// buildJSONTree recursively builds a JSON representation of the composition tree.
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
		childSuffix := nameSuffix
		if v.nameSuffix != "" {
			if nameSuffix != "" {
				childSuffix = nameSuffix + ":" + v.nameSuffix
			} else {
				childSuffix = v.nameSuffix
			}
		}

		hasPathOptions := len(v.includePaths) > 0 || len(v.excludePaths) > 0 ||
			v.detectFunc != nil
		if !hasPathOptions {
			return buildJSONTree(v.inner, childSuffix, p)
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

		pkrun.Printf(ctx, "%s%s%s%s\n", prefix, branch, effectiveName, marker)

		continuation := "│   "
		if isLast {
			continuation = "    "
		}
		pkrun.Printf(ctx, "%s%s    paths: %s\n", prefix, continuation, paths)

	case *serial:
		pkrun.Printf(ctx, "%s%s[→] Serial\n", prefix, branch)
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
		pkrun.Printf(ctx, "%s%s[⚡] Parallel\n", prefix, branch)
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
		childSuffix := nameSuffix
		if v.nameSuffix != "" {
			if nameSuffix != "" {
				childSuffix = nameSuffix + ":" + v.nameSuffix
			} else {
				childSuffix = v.nameSuffix
			}
		}

		hasPathOptions := len(v.includePaths) > 0 || len(v.excludePaths) > 0 ||
			v.detectFunc != nil
		if hasPathOptions {
			pkrun.Printf(ctx, "%s%s[📁] With paths:\n", prefix, branch)
			childPrefix := prefix
			if isLast {
				childPrefix += "    "
			} else {
				childPrefix += "│   "
			}
			if len(v.includePaths) > 0 {
				pkrun.Printf(ctx, "%s    include: %v\n", childPrefix, v.includePaths)
			}
			if len(v.excludePaths) > 0 {
				pkrun.Printf(ctx, "%s    exclude: %v\n", childPrefix, v.excludePaths)
			}
			printTree(ctx, v.inner, childPrefix, true, childSuffix, p)
		} else {
			printTree(ctx, v.inner, prefix, isLast, childSuffix, p)
		}
	}
}

// formatPaths formats a path list for display.
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
