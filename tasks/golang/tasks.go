// Package golang provides Go-related build tasks.
package golang

import (
	"slices"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
	"github.com/fredrikaverpil/pocket/tools/govulncheck"
	"github.com/goyek/goyek/v3"
)

const name = "go"

// Options defines options for a Go module within a task group.
type Options struct {
	// Skip lists task names to skip (e.g., "format", "lint", "test", "vulncheck").
	Skip []string
	// Only lists task names to run (empty = run all).
	// If non-empty, only these tasks run (Skip is ignored).
	Only []string

	// Task-specific options
	Format    FormatOptions
	Lint      LintOptions
	Test      TestOptions
	Vulncheck VulncheckOptions
}

// ShouldRun returns true if the given task should run based on Skip/Only options.
func (o Options) ShouldRun(task string) bool {
	if len(o.Only) > 0 {
		return slices.Contains(o.Only, task)
	}
	return !slices.Contains(o.Skip, task)
}

// FormatOptions defines options for the format task.
type FormatOptions struct {
	// ConfigFile overrides the default golangci-lint config file.
	ConfigFile string
}

// LintOptions defines options for the lint task.
type LintOptions struct {
	// ConfigFile overrides the default golangci-lint config file.
	ConfigFile string
}

// TestOptions defines options for the test task.
type TestOptions struct {
	// Short runs tests with -short flag.
	Short bool
	// NoRace disables the -race flag (enabled by default).
	NoRace bool
}

// VulncheckOptions defines options for the vulncheck task.
type VulncheckOptions struct {
	// placeholder for future options
}

// New creates a Go task group with the given module configuration.
func New(modules map[string]Options) pocket.TaskGroup {
	return &taskGroup{modules: modules}
}

type taskGroup struct {
	modules map[string]Options
}

func (tg *taskGroup) Name() string { return name }

func (tg *taskGroup) Modules() map[string]pocket.ModuleConfig {
	modules := make(map[string]pocket.ModuleConfig, len(tg.modules))
	for path, opts := range tg.modules {
		modules[path] = opts
	}
	return modules
}

func (tg *taskGroup) ForContext(context string) pocket.TaskGroup {
	if context == "." {
		return tg
	}
	if opts, ok := tg.modules[context]; ok {
		return &taskGroup{modules: map[string]Options{context: opts}}
	}
	return nil
}

func (tg *taskGroup) Tasks(cfg pocket.Config) []*goyek.DefinedTask {
	_ = cfg.WithDefaults()
	var tasks []*goyek.DefinedTask

	if mods := tg.modulesFor("format"); len(mods) > 0 {
		tasks = append(tasks, goyek.Define(FormatTask(mods)))
	}
	if mods := tg.modulesFor("test"); len(mods) > 0 {
		tasks = append(tasks, goyek.Define(TestTask(mods)))
	}
	if mods := tg.modulesFor("lint"); len(mods) > 0 {
		tasks = append(tasks, goyek.Define(LintTask(mods)))
	}
	if mods := tg.modulesFor("vulncheck"); len(mods) > 0 {
		tasks = append(tasks, goyek.Define(VulncheckTask(mods)))
	}

	return tasks
}

// modulesFor returns modules with their task-specific options for a given task.
func (tg *taskGroup) modulesFor(task string) map[string]Options {
	result := make(map[string]Options)
	for path, opts := range tg.modules {
		if opts.ShouldRun(task) {
			result[path] = opts
		}
	}
	return result
}

// FormatTask returns a task that formats Go code using golangci-lint fmt.
// The modules map specifies which directories to format and their options.
func FormatTask(modules map[string]Options) goyek.Task {
	return goyek.Task{
		Name:  "go-format",
		Usage: "format Go code (gofumpt, goimports, gci, golines)",
		Action: func(a *goyek.A) {
			for mod, opts := range modules {
				configPath := opts.Format.ConfigFile
				if configPath == "" {
					var err error
					configPath, err = golangcilint.ConfigPath()
					if err != nil {
						a.Fatalf("get golangci-lint config: %v", err)
					}
				}
				cmd, err := golangcilint.Command(a.Context(), "fmt", "-c", configPath, "./...")
				if err != nil {
					a.Fatalf("prepare golangci-lint: %v", err)
				}
				cmd.Dir = pocket.FromGitRoot(mod)
				if err := cmd.Run(); err != nil {
					a.Errorf("golangci-lint fmt failed in %s: %v", mod, err)
				}
			}
		},
	}
}

// TestTask returns a task that runs Go tests with race detection.
// The modules map specifies which directories to test and their options.
func TestTask(modules map[string]Options) goyek.Task {
	return goyek.Task{
		Name:  "go-test",
		Usage: "run Go tests",
		Action: func(a *goyek.A) {
			for mod, opts := range modules {
				args := []string{"test", "-v"}
				if !opts.Test.NoRace {
					args = append(args, "-race")
				}
				if opts.Test.Short {
					args = append(args, "-short")
				}
				args = append(args, "./...")
				cmd := pocket.Command(a.Context(), "go", args...)
				cmd.Dir = pocket.FromGitRoot(mod)
				if err := cmd.Run(); err != nil {
					a.Errorf("go test failed in %s: %v", mod, err)
				}
			}
		},
	}
}

// LintTask returns a task that runs golangci-lint.
// The modules map specifies which directories to lint and their options.
func LintTask(modules map[string]Options) goyek.Task {
	return goyek.Task{
		Name:  "go-lint",
		Usage: "run golangci-lint",
		Action: func(a *goyek.A) {
			for mod, opts := range modules {
				configPath := opts.Lint.ConfigFile
				if configPath == "" {
					var err error
					configPath, err = golangcilint.ConfigPath()
					if err != nil {
						a.Fatalf("get golangci-lint config: %v", err)
					}
				}
				cmd, err := golangcilint.Command(
					a.Context(),
					"run",
					"--allow-parallel-runners",
					"-c",
					configPath,
					"./...",
				)
				if err != nil {
					a.Fatalf("prepare golangci-lint: %v", err)
				}
				cmd.Dir = pocket.FromGitRoot(mod)
				if err := cmd.Run(); err != nil {
					a.Errorf("golangci-lint failed in %s: %v", mod, err)
				}
			}
		},
	}
}

// VulncheckTask returns a task that runs govulncheck.
// The modules map specifies which directories to check and their options.
func VulncheckTask(modules map[string]Options) goyek.Task {
	return goyek.Task{
		Name:  "go-vulncheck",
		Usage: "run govulncheck",
		Action: func(a *goyek.A) {
			for mod := range modules {
				cmd, err := govulncheck.Command(a.Context(), "./...")
				if err != nil {
					a.Fatalf("prepare govulncheck: %v", err)
				}
				cmd.Dir = pocket.FromGitRoot(mod)
				if err := cmd.Run(); err != nil {
					a.Errorf("govulncheck failed in %s: %v", mod, err)
				}
			}
		},
	}
}
