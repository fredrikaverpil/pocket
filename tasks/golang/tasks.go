// Package golang provides Go-related build tasks.
package golang

import (
	"context"
	"fmt"
	"slices"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
	"github.com/fredrikaverpil/pocket/tools/govulncheck"
)

// Options defines options for a Go module within a task group.
type Options struct {
	// Skip lists full task names to skip (e.g., "go-format", "go-lint", "go-test", "go-vulncheck").
	Skip []string

	// Task-specific options
	Format    FormatOptions
	Lint      LintOptions
	Test      TestOptions
	Vulncheck VulncheckOptions
}

// ShouldRun returns true if the given task should run based on the Skip list.
func (o Options) ShouldRun(taskName string) bool {
	return !slices.Contains(o.Skip, taskName)
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

// Package defines the Go task package.
var Package = pocket.TaskPackage[Options]{
	Name:   "go",
	Detect: func() []string { return pocket.DetectByFile("go.mod") },
	Tasks: []pocket.TaskDef[Options]{
		{Name: "go-format", Create: FormatTask},
		{Name: "go-lint", Create: LintTask},
		{Name: "go-test", Create: TestTask},
		{Name: "go-vulncheck", Create: VulncheckTask},
	},
}

// Auto creates a Go task group that auto-detects modules by finding go.mod files.
// The defaults parameter specifies default options for all detected modules.
// Skip patterns can be passed to exclude paths or specific tasks.
func Auto(defaults Options, opts ...pocket.SkipOption) pocket.TaskGroup {
	return Package.Auto(defaults, opts...)
}

// New creates a Go task group with explicit module configuration.
func New(modules map[string]Options) pocket.TaskGroup {
	return Package.New(modules)
}

// FormatTask returns a task that formats Go code using golangci-lint fmt.
// The modules map specifies which directories to format and their options.
func FormatTask(modules map[string]Options) *pocket.Task {
	return &pocket.Task{
		Name:  "go-format",
		Usage: "format Go code (gofumpt, goimports, gci, golines)",
		Action: func(ctx context.Context, _ map[string]string) error {
			for mod, opts := range modules {
				configPath := opts.Format.ConfigFile
				if configPath == "" {
					var err error
					configPath, err = golangcilint.ConfigPath()
					if err != nil {
						return fmt.Errorf("get golangci-lint config: %w", err)
					}
				}
				cmd, err := golangcilint.Command(ctx, "fmt", "-c", configPath, "./...")
				if err != nil {
					return fmt.Errorf("prepare golangci-lint: %w", err)
				}
				cmd.Dir = pocket.FromGitRoot(mod)
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("golangci-lint fmt failed in %s: %w", mod, err)
				}
			}
			return nil
		},
	}
}

// TestTask returns a task that runs Go tests with race detection.
// The modules map specifies which directories to test and their options.
func TestTask(modules map[string]Options) *pocket.Task {
	return &pocket.Task{
		Name:  "go-test",
		Usage: "run Go tests",
		Action: func(ctx context.Context, _ map[string]string) error {
			for mod, opts := range modules {
				args := []string{"test"}
				if pocket.IsVerbose(ctx) {
					args = append(args, "-v")
				}
				if !opts.Test.NoRace {
					args = append(args, "-race")
				}
				if opts.Test.Short {
					args = append(args, "-short")
				}
				args = append(args, "./...")
				cmd := pocket.Command(ctx, "go", args...)
				cmd.Dir = pocket.FromGitRoot(mod)
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("go test failed in %s: %w", mod, err)
				}
			}
			return nil
		},
	}
}

// LintTask returns a task that runs golangci-lint.
// The modules map specifies which directories to lint and their options.
func LintTask(modules map[string]Options) *pocket.Task {
	return &pocket.Task{
		Name:  "go-lint",
		Usage: "run golangci-lint",
		Action: func(ctx context.Context, _ map[string]string) error {
			for mod, opts := range modules {
				configPath := opts.Lint.ConfigFile
				if configPath == "" {
					var err error
					configPath, err = golangcilint.ConfigPath()
					if err != nil {
						return fmt.Errorf("get golangci-lint config: %w", err)
					}
				}
				cmd, err := golangcilint.Command(
					ctx,
					"run",
					"--allow-parallel-runners",
					"-c",
					configPath,
					"./...",
				)
				if err != nil {
					return fmt.Errorf("prepare golangci-lint: %w", err)
				}
				cmd.Dir = pocket.FromGitRoot(mod)
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("golangci-lint failed in %s: %w", mod, err)
				}
			}
			return nil
		},
	}
}

// VulncheckTask returns a task that runs govulncheck.
// The modules map specifies which directories to check and their options.
func VulncheckTask(modules map[string]Options) *pocket.Task {
	return &pocket.Task{
		Name:  "go-vulncheck",
		Usage: "run govulncheck",
		Action: func(ctx context.Context, _ map[string]string) error {
			for mod := range modules {
				cmd, err := govulncheck.Command(ctx, "./...")
				if err != nil {
					return fmt.Errorf("prepare govulncheck: %w", err)
				}
				cmd.Dir = pocket.FromGitRoot(mod)
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("govulncheck failed in %s: %w", mod, err)
				}
			}
			return nil
		},
	}
}
