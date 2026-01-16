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
	var allFuncs []*TaskDef
	pathMappings := make(map[string]*PathFilter)
	autoRunNames := make(map[string]bool)

	if cfg.AutoRun != nil {
		allFuncs = cfg.AutoRun.funcs()
		pathMappings = collectPathMappings(cfg.AutoRun)
		for _, f := range allFuncs {
			autoRunNames[f.name] = true
		}
	}

	// Create an "all" function that runs generate → AutoRun → git-diff.
	var allFunc *TaskDef
	if cfg.AutoRun != nil {
		allFunc = Task("all", "run all tasks", func(ctx context.Context) error {
			// Run generate first (unless skipped).
			if !cfg.SkipGenerate {
				if generateAllFn == nil {
					return fmt.Errorf(
						"scaffold not registered; import github.com/fredrikaverpil/pocket/internal/scaffold",
					)
				}
				if _, err := generateAllFn(&cfg); err != nil {
					return fmt.Errorf("generate: %w", err)
				}
			}

			// Run the AutoRun tree.
			if err := cfg.AutoRun.run(ctx); err != nil {
				return err
			}

			// Run git-diff at the end (unless skipped).
			if !cfg.SkipGitDiff {
				if err := Exec(ctx, "git", "diff", "--exit-code"); err != nil {
					return fmt.Errorf("uncommitted changes detected; please commit or stage your changes")
				}
			}

			return nil
		}, AsHidden())
	}

	// Add manual run functions and their path mappings.
	// Note: If the same TaskDef appears in both AutoRun and ManualRun,
	// validateNoDuplicateFuncs will error. Use WithName() to give
	// ManualRun tasks distinct names.
	for _, r := range cfg.ManualRun {
		allFuncs = append(allFuncs, r.funcs()...)
		// Collect path mappings from ManualRun so tasks are visible in subdirectories.
		for name, pf := range collectPathMappings(r) {
			pathMappings[name] = pf
		}
	}

	// Collect built-in tasks (generate and update need Config).
	builtinFuncs := builtinTasks(&cfg)

	// Validate no duplicate function names.
	if err := validateNoDuplicateFuncs(allFuncs, builtinFuncs); err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	// Call the CLI main function.
	cliMain(allFuncs, allFunc, pathMappings, autoRunNames, builtinFuncs)
}

// validateNoDuplicateFuncs checks that no two functions have the same name.
// Returns an error listing all duplicates found.
func validateNoDuplicateFuncs(funcs, builtinFuncs []*TaskDef) error {
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
	Dedup  bool `arg:"dedup"  usage:"show deduplicated items that would be skipped"`
}

// builtinTasks returns the built-in tasks that are always available.
// These include: clean, generate, git-diff, plan, update.
func builtinTasks(cfg *Config) []*TaskDef {
	return []*TaskDef{
		// plan: show the execution tree
		Task("plan", "show the execution tree and shim locations", func(ctx context.Context) error {
			opts := Options[planOptions](ctx)

			// Print AutoRun tree using Engine
			if cfg.AutoRun != nil {
				Printf(ctx, "AutoRun (./pok):\n")
				engine := NewEngine(cfg.AutoRun)
				plan, err := engine.Plan(context.Background())
				if err != nil {
					return fmt.Errorf("collect plan: %w", err)
				}
				plan.Print(ctx, opts.Hidden, opts.Dedup)
			} else {
				Printf(ctx, "AutoRun: (none)\n")
			}

			// Print shim locations
			Printf(ctx, "\nShims:\n")
			shimCfg := cfg.Shim
			if shimCfg == nil {
				shimCfg = &ShimConfig{Name: "pok", Posix: true}
			}
			root := GitRoot()
			if shimCfg.Posix {
				Printf(ctx, "  %s/%s (posix)\n", root, shimCfg.Name)
			}
			if shimCfg.Windows {
				Printf(ctx, "  %s/%s.cmd (windows)\n", root, shimCfg.Name)
			}
			if shimCfg.PowerShell {
				Printf(ctx, "  %s/%s.ps1 (powershell)\n", root, shimCfg.Name)
			}

			return nil
		}, Opts(planOptions{})),

		// clean: remove .pocket/tools and .pocket/bin directories
		Task("clean", "remove .pocket/tools and .pocket/bin directories", func(ctx context.Context) error {
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
		Task("generate", "regenerate all generated files (main.go, shim)", func(ctx context.Context) error {
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
		Task("git-diff", "fail if there are uncommitted changes", func(ctx context.Context) error {
			if err := Exec(ctx, "git", "diff", "--exit-code"); err != nil {
				return fmt.Errorf("uncommitted changes detected; please commit or stage your changes")
			}
			return nil
		}),

		// update: update pocket dependency and regenerate files
		Task("update", "update pocket dependency and regenerate files", func(ctx context.Context) error {
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
