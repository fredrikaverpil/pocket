package pocket

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// GenerateAllFunc is the function signature for scaffold.GenerateAll.
// This is set by internal/scaffold at init time to avoid import cycles.
type GenerateAllFunc func(cfg *Config) ([]string, error)

// generateAllFn is the registered scaffold function.
// Set by internal/scaffold.init() via RegisterGenerateAll.
var generateAllFn GenerateAllFunc

// RegisterGenerateAll registers the scaffold.GenerateAll function.
// This is called by internal/scaffold.init() to avoid import cycles.
func RegisterGenerateAll(fn GenerateAllFunc) {
	generateAllFn = fn
}

// RunConfig is the main entry point for running a pocket configuration.
// It parses CLI flags, discovers functions, and runs the appropriate ones.
//
// Example usage in .pocket/main.go:
//
//	func main() {
//	    pocket.RunConfig(Config)
//	}
func RunConfig(cfg Config) {
	cfg = cfg.WithDefaults()

	// Collect all functions and path mappings from AutoRun.
	var allFuncs []*FuncDef
	pathMappings := make(map[string]*PathFilter)
	autoRunNames := make(map[string]bool)

	if cfg.AutoRun != nil {
		allFuncs = cfg.AutoRun.funcs()
		pathMappings = CollectPathMappings(cfg.AutoRun)
		for _, f := range allFuncs {
			autoRunNames[f.name] = true
		}
	}

	// Create an "all" function that runs the entire AutoRun tree.
	var allFunc *FuncDef
	if cfg.AutoRun != nil {
		allFunc = Func("all", "run all tasks", func(ctx context.Context) error {
			return cfg.AutoRun.run(ctx)
		})
	}

	// Add manual run functions (if any - ManualRun is []Runnable in old Config).
	for _, r := range cfg.ManualRun {
		allFuncs = append(allFuncs, r.funcs()...)
	}

	// Collect built-in tasks (generate and update need Config).
	builtinFuncs := builtinTasks(&cfg)

	// Validate no duplicate function names.
	if err := validateNoDuplicateFuncs(allFuncs, builtinFuncs); err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	// Call the CLI main function.
	Main(allFuncs, allFunc, nil, pathMappings, autoRunNames, builtinFuncs)
}

// validateNoDuplicateFuncs checks that no two functions have the same name.
// Returns an error listing all duplicates found.
func validateNoDuplicateFuncs(funcs, builtinFuncs []*FuncDef) error {
	seen := make(map[string]bool)
	var duplicates []string

	// Check user functions.
	for _, f := range funcs {
		if seen[f.name] {
			duplicates = append(duplicates, f.name)
		}
		seen[f.name] = true
	}

	// Check built-in functions don't conflict with user functions.
	for _, f := range builtinFuncs {
		if seen[f.name] {
			duplicates = append(duplicates, f.name+" (conflicts with builtin)")
		}
	}

	if len(duplicates) > 0 {
		return fmt.Errorf("duplicate function names: %s", strings.Join(duplicates, ", "))
	}
	return nil
}

// planOptions configures the plan command.
type planOptions struct {
	Hidden bool `arg:"hidden" usage:"show hidden functions (e.g., install tasks)"`
}

// builtinTasks returns the built-in tasks that are always available.
// These include: clean, generate, git-diff, plan, update.
func builtinTasks(cfg *Config) []*FuncDef {
	return []*FuncDef{
		// plan: show the execution tree
		Func("plan", "show the execution tree and shim locations", func(ctx context.Context) error {
			opts := Options[planOptions](ctx)
			printPlan(ctx, cfg, opts.Hidden)
			return nil
		}).With(planOptions{}),

		// clean: remove .pocket/tools and .pocket/bin directories
		Func("clean", "remove .pocket/tools and .pocket/bin directories", func(ctx context.Context) error {
			toolsDir := FromToolsDir()
			if _, err := os.Stat(toolsDir); err == nil {
				if err := os.RemoveAll(toolsDir); err != nil {
					return fmt.Errorf("remove tools dir: %w", err)
				}
				Printf(ctx, "Removed %s\n", toolsDir)
			}
			binDir := FromBinDir()
			if _, err := os.Stat(binDir); err == nil {
				if err := os.RemoveAll(binDir); err != nil {
					return fmt.Errorf("remove bin dir: %w", err)
				}
				Printf(ctx, "Removed %s\n", binDir)
			}
			return nil
		}),

		// generate: regenerate all generated files (main.go, shim)
		Func("generate", "regenerate all generated files (main.go, shim)", func(ctx context.Context) error {
			if generateAllFn == nil {
				return fmt.Errorf("scaffold not registered; import github.com/fredrikaverpil/pocket/internal/scaffold")
			}
			shimPaths, err := generateAllFn(cfg)
			if err != nil {
				return err
			}
			if Verbose(ctx) {
				Printf(ctx, "Generated .pocket/main.go and shims:\n  %s\n", strings.Join(shimPaths, "\n  "))
			} else {
				Printf(ctx, "Generated .pocket/main.go and %d shim(s)\n", len(shimPaths))
			}
			return nil
		}),

		// git-diff: fail if there are uncommitted changes
		Func("git-diff", "fail if there are uncommitted changes", func(ctx context.Context) error {
			if err := Exec(ctx, "git", "diff", "--exit-code"); err != nil {
				return fmt.Errorf("uncommitted changes detected; please commit or stage your changes")
			}
			return nil
		}),

		// update: update pocket dependency and regenerate files
		Func("update", "update pocket dependency and regenerate files", func(ctx context.Context) error {
			verbose := Verbose(ctx)
			pocketDir := FromPocketDir()

			// Update pocket dependency with GOPROXY=direct to bypass proxy cache.
			if verbose {
				Println(ctx, "Updating github.com/fredrikaverpil/pocket@latest")
			}
			cmd := Command(ctx, "go", "get", "-u", "github.com/fredrikaverpil/pocket@latest")
			cmd.Dir = pocketDir
			cmd.Env = append(cmd.Env, "GOPROXY=direct")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("go get -u: %w", err)
			}

			// Run go mod tidy
			if verbose {
				Println(ctx, "Running go mod tidy")
			}
			if err := ExecIn(ctx, pocketDir, "go", "mod", "tidy"); err != nil {
				return fmt.Errorf("go mod tidy: %w", err)
			}

			// Regenerate all files
			if verbose {
				Println(ctx, "Regenerating files")
			}
			if generateAllFn == nil {
				return fmt.Errorf("scaffold not registered; import github.com/fredrikaverpil/pocket/internal/scaffold")
			}
			if _, err := generateAllFn(cfg); err != nil {
				return err
			}

			if verbose {
				Println(ctx, "Done!")
			}
			return nil
		}),
	}
}

// printPlan prints the execution tree and shim locations.
func printPlan(ctx context.Context, cfg *Config, showHidden bool) {
	// Print AutoRun tree
	if cfg.AutoRun != nil {
		Printf(ctx, "AutoRun (./pok):\n")
		printTree(ctx, cfg.AutoRun, "  ", true, showHidden)
	} else {
		Printf(ctx, "AutoRun: (none)\n")
	}

	// Print shim locations
	Printf(ctx, "\nShims:\n")
	if cfg.Shim == nil {
		Printf(ctx, "  (using defaults)\n")
		cfg = &Config{Shim: cfg.Shim}
		*cfg = cfg.WithDefaults()
	}
	shim := cfg.Shim
	if shim == nil {
		shim = &ShimConfig{Name: "pok", Posix: true}
	}
	root := GitRoot()
	if shim.Posix {
		Printf(ctx, "  %s/%s (posix)\n", root, shim.Name)
	}
	if shim.Windows {
		Printf(ctx, "  %s/%s.cmd (windows)\n", root, shim.Name)
	}
	if shim.PowerShell {
		Printf(ctx, "  %s/%s.ps1 (powershell)\n", root, shim.Name)
	}
}

// printTree recursively prints a Runnable tree with indentation.
func printTree(ctx context.Context, r Runnable, indent string, last, showHidden bool) {
	// Determine the connector
	connector := "├── "
	if last {
		connector = "└── "
	}
	childIndent := indent + "│   "
	if last {
		childIndent = indent + "    "
	}

	switch v := r.(type) {
	case *FuncDef:
		// Skip hidden functions unless showHidden is true
		if v.hidden && !showHidden {
			return
		}
		label := v.name
		if v.hidden {
			label += " (hidden)"
		}
		if v.usage != "" {
			label += " - " + v.usage
		}
		Printf(ctx, "%s%s%s\n", indent, connector, label)
		// If FuncDef has a body (composition), print it
		if v.body != nil {
			printTree(ctx, v.body, childIndent, true, showHidden)
		}
	case *serial:
		// Filter items if not showing hidden
		items := v.items
		if !showHidden {
			items = filterVisible(items)
		}
		if len(items) == 0 {
			return
		}
		Printf(ctx, "%s%sSerial:\n", indent, connector)
		for i, item := range items {
			printTree(ctx, item, childIndent, i == len(items)-1, showHidden)
		}
	case *parallel:
		// Filter items if not showing hidden
		items := v.items
		if !showHidden {
			items = filterVisible(items)
		}
		if len(items) == 0 {
			return
		}
		Printf(ctx, "%s%sParallel:\n", indent, connector)
		for i, item := range items {
			printTree(ctx, item, childIndent, i == len(items)-1, showHidden)
		}
	case *PathFilter:
		paths := describePathFilter(v)
		Printf(ctx, "%s%sPaths(%s):\n", indent, connector, paths)
		printTree(ctx, v.inner, childIndent, true, showHidden)
	default:
		Printf(ctx, "%s%s(unknown: %T)\n", indent, connector, r)
	}
}

// filterVisible returns only visible (non-hidden) runnables.
func filterVisible(items []Runnable) []Runnable {
	result := make([]Runnable, 0, len(items))
	for _, item := range items {
		if f, ok := item.(*FuncDef); ok && f.hidden {
			continue
		}
		result = append(result, item)
	}
	return result
}

// describePathFilter returns a human-readable description of path filtering.
func describePathFilter(p *PathFilter) string {
	var parts []string
	if p.detect != nil {
		detected := p.detect()
		if len(detected) > 0 {
			parts = append(parts, "detect: "+strings.Join(detected, ", "))
		} else {
			parts = append(parts, "detect: (none)")
		}
	}
	if len(p.include) > 0 {
		var patterns []string
		for _, re := range p.include {
			patterns = append(patterns, re.String())
		}
		parts = append(parts, "in: "+strings.Join(patterns, ", "))
	}
	if len(p.exclude) > 0 {
		var patterns []string
		for _, re := range p.exclude {
			patterns = append(patterns, re.String())
		}
		parts = append(parts, "except: "+strings.Join(patterns, ", "))
	}
	if len(parts) == 0 {
		return "all"
	}
	return strings.Join(parts, "; ")
}
