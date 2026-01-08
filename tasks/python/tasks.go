// Package python provides Python-related build tasks using ruff and mypy.
package python

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/mypy"
	"github.com/fredrikaverpil/pocket/tools/ruff"
)

// Tasks returns a Runnable that executes all Python tasks.
// Tasks auto-detect Python projects by finding pyproject.toml, setup.py, or setup.cfg.
// Use pocket.P(python.Tasks()).Detect() to enable path filtering.
func Tasks() pocket.Runnable {
	return &pyTasks{
		format:    FormatTask(),
		lint:      LintTask(),
		typecheck: TypecheckTask(),
	}
}

// pyTasks is the Runnable for Python tasks that also implements Detectable.
type pyTasks struct {
	format    *pocket.Task
	lint      *pocket.Task
	typecheck *pocket.Task
}

// Run executes all Python tasks.
// Format runs first, then lint and typecheck run in parallel.
func (p *pyTasks) Run(ctx context.Context) error {
	if err := p.format.Run(ctx); err != nil {
		return err
	}
	return pocket.Deps(ctx, p.lint, p.typecheck)
}

// Tasks returns all Python tasks.
func (p *pyTasks) Tasks() []*pocket.Task {
	return []*pocket.Task{p.format, p.lint, p.typecheck}
}

// DefaultDetect returns a function that detects Python project directories.
func (p *pyTasks) DefaultDetect() func() []string {
	return detectModules
}

// detectModules returns directories containing Python project files.
func detectModules() []string {
	return pocket.DetectByFile("pyproject.toml", "setup.py", "setup.cfg")
}

// FormatTask returns a task that formats Python files using ruff format.
func FormatTask() *pocket.Task {
	return &pocket.Task{
		Name:  "py-format",
		Usage: "format Python files",
		Action: func(ctx context.Context, _ map[string]string) error {
			configPath, err := ruff.ConfigPath()
			if err != nil {
				return fmt.Errorf("get ruff config: %w", err)
			}

			for _, dir := range detectModules() {
				if err := ruff.Run(ctx, "format", "--config", configPath, dir); err != nil {
					return fmt.Errorf("ruff format failed in %s: %w", dir, err)
				}
			}
			return nil
		},
	}
}

// LintTask returns a task that lints Python files using ruff check.
func LintTask() *pocket.Task {
	return &pocket.Task{
		Name:  "py-lint",
		Usage: "lint Python files",
		Action: func(ctx context.Context, _ map[string]string) error {
			configPath, err := ruff.ConfigPath()
			if err != nil {
				return fmt.Errorf("get ruff config: %w", err)
			}

			for _, dir := range detectModules() {
				if err := ruff.Run(ctx, "check", "--config", configPath, dir); err != nil {
					return fmt.Errorf("ruff check failed in %s: %w", dir, err)
				}
			}
			return nil
		},
	}
}

// TypecheckTask returns a task that type-checks Python files using mypy.
func TypecheckTask() *pocket.Task {
	return &pocket.Task{
		Name:  "py-typecheck",
		Usage: "type-check Python files",
		Action: func(ctx context.Context, _ map[string]string) error {
			for _, dir := range detectModules() {
				if err := mypy.Run(ctx, dir); err != nil {
					return fmt.Errorf("mypy failed in %s: %w", dir, err)
				}
			}
			return nil
		},
	}
}
