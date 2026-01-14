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

	// Call the CLI main function.
	Main(allFuncs, allFunc, nil, pathMappings, autoRunNames, builtinFuncs)
}

// RunConfig2 is the main entry point for running a pocket v2 configuration.
// It supports the new Cmd-based manual run.
//
// Example usage in .pocket/main.go:
//
//	func main() {
//	    pocket.RunConfig2(Config)
//	}
func RunConfig2(cfg Config2) {
	cfg = cfg.WithDefaults2()

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

	// Collect built-in tasks (generate and update need Config).
	// Convert Config2 to Config for built-in tasks that need it.
	legacyCfg := Config{
		AutoRun:     cfg.AutoRun,
		Shim:        cfg.Shim,
		SkipGitDiff: cfg.SkipGitDiff,
	}
	builtinFuncs := builtinTasks(&legacyCfg)

	// Call the CLI main function with commands.
	Main(allFuncs, allFunc, cfg.ManualRun, pathMappings, autoRunNames, builtinFuncs)
}

// builtinTasks returns the built-in tasks that are always available.
// These include: clean, generate, git-diff, update.
func builtinTasks(cfg *Config) []*FuncDef {
	return []*FuncDef{
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

			// Update pocket dependency
			if verbose {
				Println(ctx, "Updating github.com/fredrikaverpil/pocket@latest")
			}
			if err := ExecIn(ctx, pocketDir, "go", "get", "-u", "github.com/fredrikaverpil/pocket@latest"); err != nil {
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
