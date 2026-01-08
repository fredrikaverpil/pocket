// Package python provides Python-related build tasks using ruff and mypy.
package python

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/mypy"
	"github.com/fredrikaverpil/pocket/tools/ruff"
)

// Tasks returns a Runnable that executes all Python tasks in order.
// Tasks auto-detect Python projects by finding pyproject.toml, setup.py, or setup.cfg.
func Tasks() pocket.Runnable {
	return pocket.Serial(
		FormatTask(),
		LintTask(),
		TypecheckTask(),
	)
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
