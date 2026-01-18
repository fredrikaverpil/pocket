package pocket

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

// GenerateAllFunc is the function signature for scaffold.GenerateAll.
// This is set by internal/scaffold at init time to avoid import cycles.
// Accepts ConfigPlan to reuse cached ModuleDirectories (avoids re-walking trees).
type GenerateAllFunc func(plan *ConfigPlan) ([]string, error)

// generateAllFn is the registered scaffold function.
// Set by internal/scaffold.init() via RegisterGenerateAll.
var generateAllFn GenerateAllFunc

// RegisterGenerateAll registers the scaffold.GenerateAll function.
// This is called by internal/scaffold.init() to avoid import cycles.
func RegisterGenerateAll(fn GenerateAllFunc) {
	generateAllFn = fn
}

// ConfigPlan holds all collected data from walking a Config's task trees.
// This is the result of the planning phase, before CLI execution.
type ConfigPlan struct {
	// Tasks collected from AutoRun and ManualRun trees
	Tasks []*TaskDef
	// AutoRunNames tracks which tasks are from AutoRun (vs ManualRun)
	AutoRunNames map[string]bool
	// PathMappings maps task names to their PathFilter for visibility
	PathMappings map[string]*PathFilter
	// AllTask is the hidden task that runs the full AutoRun tree
	AllTask *TaskDef
	// BuiltinTasks are always-available tasks (plan, clean, generate, etc.)
	BuiltinTasks []*TaskDef
	// ModuleDirectories are all directories where shims should be generated
	ModuleDirectories []string
	// Config is the original configuration (for builtin tasks that need it)
	Config *Config
}

// BuildConfigPlan walks the Config's task trees and collects all data needed
// for CLI execution. This is the single point where tree walking happens.
func BuildConfigPlan(cfg Config) *ConfigPlan {
	plan := &ConfigPlan{
		Tasks:        make([]*TaskDef, 0),
		AutoRunNames: make(map[string]bool),
		PathMappings: make(map[string]*PathFilter),
		Config:       &cfg,
	}

	// Collect module directories from all trees
	moduleDirSet := make(map[string]bool)
	moduleDirSet["."] = true // Always include root

	// Phase 1: Walk AutoRun tree
	if cfg.AutoRun != nil {
		engine := NewEngine(cfg.AutoRun)
		if execPlan, err := engine.Plan(context.Background()); err == nil {
			plan.Tasks = execPlan.TaskDefs()
			plan.PathMappings = execPlan.PathMappings()
			for _, dir := range execPlan.ModuleDirectories() {
				moduleDirSet[dir] = true
			}
		}
		for _, f := range plan.Tasks {
			plan.AutoRunNames[f.name] = true
		}
	}

	// Phase 2: Create the "all" task (runs generate → AutoRun → git-diff)
	if cfg.AutoRun != nil {
		plan.AllTask = Task("all", "run all tasks", func(ctx context.Context) error {
			if !cfg.SkipGenerate {
				if generateAllFn == nil {
					return fmt.Errorf(
						"scaffold not registered; import github.com/fredrikaverpil/pocket/internal/scaffold",
					)
				}
				configPlan := GetConfigPlan(ctx)
				if _, err := generateAllFn(configPlan); err != nil {
					return fmt.Errorf("generate: %w", err)
				}
			}
			if err := cfg.AutoRun.run(ctx); err != nil {
				return err
			}
			if !cfg.SkipGitDiff {
				if err := Exec(ctx, "git", "diff", "--exit-code"); err != nil {
					return fmt.Errorf("uncommitted changes detected; please commit or stage your changes")
				}
			}
			return nil
		}, AsHidden())
	}

	// Phase 3: Walk ManualRun trees
	for _, r := range cfg.ManualRun {
		engine := NewEngine(r)
		if execPlan, err := engine.Plan(context.Background()); err == nil {
			plan.Tasks = append(plan.Tasks, execPlan.TaskDefs()...)
			for name, pf := range execPlan.PathMappings() {
				plan.PathMappings[name] = pf
			}
			for _, dir := range execPlan.ModuleDirectories() {
				moduleDirSet[dir] = true
			}
		}
	}

	// Convert module directories set to sorted slice
	plan.ModuleDirectories = make([]string, 0, len(moduleDirSet))
	for dir := range moduleDirSet {
		plan.ModuleDirectories = append(plan.ModuleDirectories, dir)
	}
	slices.Sort(plan.ModuleDirectories)

	// Phase 4: Add built-in tasks
	plan.BuiltinTasks = builtinTasks(&cfg)

	return plan
}

// Validate checks the ConfigPlan for errors (e.g., duplicate task names).
func (p *ConfigPlan) Validate() error {
	seen := make(map[string]bool)
	var duplicates []string

	for _, f := range p.Tasks {
		if seen[f.name] {
			duplicates = append(duplicates, f.name)
		}
		seen[f.name] = true
	}

	for _, f := range p.BuiltinTasks {
		if seen[f.name] {
			duplicates = append(duplicates, f.name+" (conflicts with builtin)")
		}
	}

	if len(duplicates) > 0 {
		return fmt.Errorf("duplicate function names: %s", strings.Join(duplicates, ", "))
	}
	return nil
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

	// Phase 1: Build the plan (walks all trees once)
	plan := BuildConfigPlan(cfg)

	// Phase 2: Validate
	if err := plan.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	// Phase 3: Run CLI
	cliMain(plan)
}

// planOptions configures the plan command.
type planOptions struct {
	Hidden  bool   `arg:"hidden"  usage:"show hidden functions (e.g., install tasks)"`
	Dedup   bool   `arg:"dedup"   usage:"show deduplicated items that would be skipped"`
	JSON    bool   `arg:"json"    usage:"output as JSON for machine consumption"`
	Outfile string `arg:"outfile" usage:"write JSON output to file (implies -json)"`
}

// builtinTasks returns the built-in tasks that are always available.
// These include: clean, generate, git-diff, plan, update.
func builtinTasks(cfg *Config) []*TaskDef {
	return []*TaskDef{
		// plan: show the execution tree
		Task("plan", "show the execution tree and shim locations", func(ctx context.Context) error {
			opts := Options[planOptions](ctx)

			// JSON output mode (-json or -outfile)
			if opts.JSON || opts.Outfile != "" {
				plan, err := BuildIntrospectPlan(*cfg)
				if err != nil {
					return fmt.Errorf("plan: %w", err)
				}
				data, err := json.MarshalIndent(plan, "", "  ")
				if err != nil {
					return fmt.Errorf("plan: marshal: %w", err)
				}

				// Validate the JSON output by unmarshaling it back
				var validate IntrospectPlan
				if err := json.Unmarshal(data, &validate); err != nil {
					return fmt.Errorf("plan: validate: %w", err)
				}

				// Write to file or stdout
				if opts.Outfile != "" {
					if err := os.WriteFile(opts.Outfile, append(data, '\n'), 0o644); err != nil {
						return fmt.Errorf("plan: write %s: %w", opts.Outfile, err)
					}
					Printf(ctx, "Wrote %s\n", opts.Outfile)
				} else {
					Printf(ctx, "%s\n", data)
				}
				return nil
			}

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
		}, Opts(planOptions{}), AsSilent()),

		// clean: remove .pocket/tools, .pocket/bin, and .pocket/venvs directories
		Task(
			"clean",
			"remove .pocket/tools, .pocket/bin, and .pocket/venvs directories",
			func(ctx context.Context) error {
				for _, dir := range []string{FromToolsDir(), FromBinDir(), FromPocketDir("venvs")} {
					if _, err := os.Stat(dir); err == nil {
						if err := os.RemoveAll(dir); err != nil {
							return fmt.Errorf("remove %s: %w", dir, err)
						}
						Printf(ctx, "Removed %s\n", dir)
					}
				}
				return nil
			},
		),

		// generate: regenerate all generated files (main.go, shim)
		Task("generate", "regenerate all generated files (main.go, shim)", func(ctx context.Context) error {
			if generateAllFn == nil {
				return fmt.Errorf("scaffold not registered; import github.com/fredrikaverpil/pocket/internal/scaffold")
			}
			configPlan := GetConfigPlan(ctx)
			shimPaths, err := generateAllFn(configPlan)
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
			out := GetOutput(ctx)
			cmd.Stdout = out.Stdout
			cmd.Stderr = out.Stderr
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
			configPlan := GetConfigPlan(ctx)
			if _, err := generateAllFn(configPlan); err != nil {
				return err
			}

			if verbose {
				Println(ctx, "Done!")
			}
			return nil
		}),
	}
}
