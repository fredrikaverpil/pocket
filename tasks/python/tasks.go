// Package python provides Python-related build tasks using ruff and mypy.
package python

import (
	"context"
	"fmt"
	"slices"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/mypy"
	"github.com/fredrikaverpil/pocket/tools/ruff"
)

// Options defines options for a Python module within a task group.
type Options struct {
	// Skip lists full task names to skip (e.g., "py-format", "py-lint", "py-typecheck").
	Skip []string

	// Task-specific options
	Format    FormatOptions
	Lint      LintOptions
	Typecheck TypecheckOptions
}

// ShouldRun returns true if the given task should run based on the Skip list.
func (o Options) ShouldRun(taskName string) bool {
	return !slices.Contains(o.Skip, taskName)
}

// FormatOptions defines options for the format task.
type FormatOptions struct {
	// ConfigFile overrides the default ruff config file.
	ConfigFile string
}

// LintOptions defines options for the lint task.
type LintOptions struct {
	// ConfigFile overrides the default ruff config file.
	ConfigFile string
}

// TypecheckOptions defines options for the typecheck task.
type TypecheckOptions struct {
	// placeholder for future options
}

// Group defines the Python task group.
var Group = pocket.TaskGroupDef[Options]{
	Name:   "python",
	Detect: func() []string { return pocket.DetectByFile("pyproject.toml", "setup.py", "setup.cfg") },
	Tasks: []pocket.TaskDef[Options]{
		{Name: "py-format", Create: FormatTask},
		{Name: "py-lint", Create: LintTask},
		{Name: "py-typecheck", Create: TypecheckTask},
	},
}

// Auto creates a Python task group that auto-detects modules by finding
// pyproject.toml, setup.py, or setup.cfg files.
// The defaults parameter specifies default options for all detected modules.
// Skip patterns can be passed to exclude paths or specific tasks.
func Auto(defaults Options, opts ...pocket.SkipOption) pocket.TaskGroup {
	return Group.Auto(defaults, opts...)
}

// New creates a Python task group with explicit module configuration.
func New(modules map[string]Options) pocket.TaskGroup {
	return Group.New(modules)
}

// FormatTask returns a task that formats Python files using ruff format.
// The modules map specifies which directories to format and their options.
func FormatTask(modules map[string]Options) *pocket.Task {
	return &pocket.Task{
		Name:  "py-format",
		Usage: "format Python files",
		Action: func(ctx context.Context, _ map[string]string) error {
			for mod, opts := range modules {
				configPath := opts.Format.ConfigFile
				if configPath == "" {
					var err error
					configPath, err = ruff.ConfigPath()
					if err != nil {
						return fmt.Errorf("get ruff config: %w", err)
					}
				}
				if err := ruff.Run(ctx, "format", "--config", configPath, mod); err != nil {
					return fmt.Errorf("ruff format failed in %s: %w", mod, err)
				}
			}
			return nil
		},
	}
}

// LintTask returns a task that lints Python files using ruff check.
// The modules map specifies which directories to lint and their options.
func LintTask(modules map[string]Options) *pocket.Task {
	return &pocket.Task{
		Name:  "py-lint",
		Usage: "lint Python files",
		Action: func(ctx context.Context, _ map[string]string) error {
			for mod, opts := range modules {
				configPath := opts.Lint.ConfigFile
				if configPath == "" {
					var err error
					configPath, err = ruff.ConfigPath()
					if err != nil {
						return fmt.Errorf("get ruff config: %w", err)
					}
				}
				if err := ruff.Run(ctx, "check", "--config", configPath, mod); err != nil {
					return fmt.Errorf("ruff check failed in %s: %w", mod, err)
				}
			}
			return nil
		},
	}
}

// TypecheckTask returns a task that type-checks Python files using mypy.
// The modules map specifies which directories to check and their options.
func TypecheckTask(modules map[string]Options) *pocket.Task {
	return &pocket.Task{
		Name:  "py-typecheck",
		Usage: "type-check Python files",
		Action: func(ctx context.Context, _ map[string]string) error {
			for mod := range modules {
				if err := mypy.Run(ctx, mod); err != nil {
					return fmt.Errorf("mypy failed in %s: %w", mod, err)
				}
			}
			return nil
		},
	}
}
