// Package python provides Python-related build tasks using ruff and mypy.
package python

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/mypy"
	"github.com/fredrikaverpil/pocket/tools/ruff"
)

// TasksOption configures the python task group.
type TasksOption func(*tasksConfig)

type tasksConfig struct {
	format FormatOptions
	lint   LintOptions
}

// WithFormat sets options for the py-format task.
func WithFormat(opts FormatOptions) TasksOption {
	return func(c *tasksConfig) { c.format = opts }
}

// WithLint sets options for the py-lint task.
func WithLint(opts LintOptions) TasksOption {
	return func(c *tasksConfig) { c.lint = opts }
}

// Tasks returns a Runnable that executes all Python tasks.
// Use pocket.Paths(python.Tasks()).DetectBy(python.Detect()) to enable path filtering.
//
// Execution order: format runs first, then lint and typecheck run in parallel.
//
// Example with options:
//
//	pocket.Paths(python.Tasks(
//	    python.WithFormat(python.FormatOptions{RuffConfig: "ruff.toml"}),
//	)).DetectBy(python.Detect())
func Tasks(opts ...TasksOption) pocket.Runnable {
	var cfg tasksConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	format := FormatTask().WithOptions(cfg.format)
	lint := LintTask().WithOptions(cfg.lint)
	typecheck := TypecheckTask()

	return pocket.Serial(format, pocket.Parallel(lint, typecheck))
}

// Detect returns a detection function that finds Python projects.
// It detects directories containing pyproject.toml, setup.py, or setup.cfg.
//
// Usage:
//
//	pocket.Paths(python.Tasks()).DetectBy(python.Detect())
func Detect() func() []string {
	return func() []string {
		return pocket.DetectByFile("pyproject.toml", "setup.py", "setup.cfg")
	}
}

// FormatOptions configures the py-format task.
type FormatOptions struct {
	RuffConfig string `usage:"path to ruff config file"`
}

// FormatTask returns a task that formats Python files using ruff format.
// Use WithOptions to set project-level configuration.
func FormatTask() *pocket.Task {
	return pocket.NewTask("py-format", "format Python files", formatAction)
}

// formatAction is the action for the py-format task.
func formatAction(ctx context.Context, tc *pocket.TaskContext) error {
	opts := pocket.GetOptions[FormatOptions](tc)
	configPath := opts.RuffConfig
	if configPath == "" {
		var err error
		configPath, err = ruff.Tool.ConfigPath()
		if err != nil {
			return fmt.Errorf("get ruff config: %w", err)
		}
	}

	if err := ruff.Tool.Exec(ctx, tc, "format", "--config", configPath, tc.Path); err != nil {
		return fmt.Errorf("ruff format failed in %s: %w", tc.Path, err)
	}
	return nil
}

// LintOptions configures the py-lint task.
type LintOptions struct {
	RuffConfig string `usage:"path to ruff config file"`
}

// LintTask returns a task that lints Python files using ruff check.
// Use WithOptions to set project-level configuration.
func LintTask() *pocket.Task {
	return pocket.NewTask("py-lint", "lint Python files", lintAction)
}

// lintAction is the action for the py-lint task.
func lintAction(ctx context.Context, tc *pocket.TaskContext) error {
	opts := pocket.GetOptions[LintOptions](tc)
	configPath := opts.RuffConfig
	if configPath == "" {
		var err error
		configPath, err = ruff.Tool.ConfigPath()
		if err != nil {
			return fmt.Errorf("get ruff config: %w", err)
		}
	}

	if err := ruff.Tool.Exec(ctx, tc, "check", "--config", configPath, tc.Path); err != nil {
		return fmt.Errorf("ruff check failed in %s: %w", tc.Path, err)
	}
	return nil
}

// TypecheckTask returns a task that type-checks Python files using mypy.
func TypecheckTask() *pocket.Task {
	return pocket.NewTask("py-typecheck", "type-check Python files", typecheckAction)
}

// typecheckAction is the action for the py-typecheck task.
func typecheckAction(ctx context.Context, tc *pocket.TaskContext) error {
	if err := mypy.Tool.Exec(ctx, tc, tc.Path); err != nil {
		return fmt.Errorf("mypy failed in %s: %w", tc.Path, err)
	}
	return nil
}
