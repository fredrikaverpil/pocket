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

// Config defines the configuration for the Go task group.
type Config struct {
	// Modules maps context paths to module options.
	// The key is the path relative to the git root (use "." for root).
	Modules map[string]Options
}

// Options defines options for a Go module.
type Options struct {
	// Skip lists task names to skip (e.g., "format", "lint", "test", "vulncheck").
	Skip []string
	// Only lists task names to run (empty = run all).
	// If non-empty, only these tasks run (Skip is ignored).
	Only []string
}

// ShouldRun returns true if the given task should run based on Skip/Only options.
func (o Options) ShouldRun(task string) bool {
	if len(o.Only) > 0 {
		return slices.Contains(o.Only, task)
	}
	return !slices.Contains(o.Skip, task)
}

// New creates a Go task group with the given configuration.
func New(cfg Config) pocket.TaskGroup {
	return &taskGroup{config: cfg}
}

type taskGroup struct {
	config Config
}

func (tg *taskGroup) Name() string { return name }

func (tg *taskGroup) Modules() map[string]pocket.ModuleConfig {
	modules := make(map[string]pocket.ModuleConfig, len(tg.config.Modules))
	for path, opts := range tg.config.Modules {
		modules[path] = opts
	}
	return modules
}

func (tg *taskGroup) ForContext(context string) pocket.TaskGroup {
	if context == "." {
		return tg
	}
	if opts, ok := tg.config.Modules[context]; ok {
		return &taskGroup{config: Config{
			Modules: map[string]Options{context: opts},
		}}
	}
	return nil
}

func (tg *taskGroup) Tasks(cfg pocket.Config) []*goyek.DefinedTask {
	_ = cfg.WithDefaults()
	var tasks []*goyek.DefinedTask

	if modules := pocket.ModulesFor(tg, "format"); len(modules) > 0 {
		tasks = append(tasks, goyek.Define(FormatTask(modules)))
	}
	if modules := pocket.ModulesFor(tg, "test"); len(modules) > 0 {
		tasks = append(tasks, goyek.Define(TestTask(modules)))
	}
	if modules := pocket.ModulesFor(tg, "lint"); len(modules) > 0 {
		tasks = append(tasks, goyek.Define(LintTask(modules)))
	}
	if modules := pocket.ModulesFor(tg, "vulncheck"); len(modules) > 0 {
		tasks = append(tasks, goyek.Define(VulncheckTask(modules)))
	}

	return tasks
}

// FormatTask returns a task that formats Go code using golangci-lint fmt.
func FormatTask(modules []string) goyek.Task {
	return goyek.Task{
		Name:  "go-format",
		Usage: "format Go code (gofumpt, goimports, gci, golines)",
		Action: func(a *goyek.A) {
			configPath, err := golangcilint.ConfigPath()
			if err != nil {
				a.Fatalf("get golangci-lint config: %v", err)
			}
			for _, mod := range modules {
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
func TestTask(modules []string) goyek.Task {
	return goyek.Task{
		Name:  "go-test",
		Usage: "run Go tests",
		Action: func(a *goyek.A) {
			for _, mod := range modules {
				cmd := pocket.Command(a.Context(), "go", "test", "-v", "-race", "./...")
				cmd.Dir = pocket.FromGitRoot(mod)
				if err := cmd.Run(); err != nil {
					a.Errorf("go test failed in %s: %v", mod, err)
				}
			}
		},
	}
}

// LintTask returns a task that runs golangci-lint.
func LintTask(modules []string) goyek.Task {
	return goyek.Task{
		Name:  "go-lint",
		Usage: "run golangci-lint",
		Action: func(a *goyek.A) {
			configPath, err := golangcilint.ConfigPath()
			if err != nil {
				a.Fatalf("get golangci-lint config: %v", err)
			}
			for _, mod := range modules {
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
func VulncheckTask(modules []string) goyek.Task {
	return goyek.Task{
		Name:  "go-vulncheck",
		Usage: "run govulncheck",
		Action: func(a *goyek.A) {
			for _, mod := range modules {
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
