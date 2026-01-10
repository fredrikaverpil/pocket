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
// Use pocket.AutoDetect(python.Tasks()) to enable path filtering.
//
// Execution order: format runs first, then lint and typecheck run in parallel.
func Tasks() pocket.Runnable {
	format := FormatTask()
	lint := LintTask()
	typecheck := TypecheckTask()

	return pocket.NewTaskGroup(format, lint, typecheck).
		RunWith(func(ctx context.Context) error {
			// Format must run first.
			if err := format.Run(ctx); err != nil {
				return err
			}
			// Lint and typecheck can run in parallel.
			return pocket.Parallel(lint, typecheck).Run(ctx)
		}).
		DetectByFile("pyproject.toml", "setup.py", "setup.cfg")
}

// FormatOptions configures the py-format task.
type FormatOptions struct {
	RuffConfig string `usage:"path to ruff config file"`
}

// formatCheck runs ruff format --check --diff to see if formatting is needed.
// Returns true if files need formatting, along with the diff output.
func formatCheck(ctx context.Context, configPath, dir string) (needsFormat bool, output []byte, err error) {
	cmd, err := ruff.Command(ctx, "format", "--check", "--diff", "--config", configPath, dir)
	if err != nil {
		return false, nil, fmt.Errorf("prepare ruff: %w", err)
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	output, checkErr := cmd.CombinedOutput()
	return checkErr != nil, output, nil
}

// FormatTask returns a task that formats Python files using ruff format.
// Optional defaults can be passed to set project-level configuration.
func FormatTask(defaults ...FormatOptions) *pocket.Task {
	return &pocket.Task{
		Name:    "py-format",
		Usage:   "format Python files",
		Options: pocket.FirstOrZero(defaults...),
		Action: func(ctx context.Context, rc *pocket.RunContext) error {
			opts := pocket.GetOptions[FormatOptions](rc)
			configPath := opts.RuffConfig
			if configPath == "" {
				var err error
				configPath, err = ruff.ConfigPath()
				if err != nil {
					return fmt.Errorf("get ruff config: %w", err)
				}
			}
			return rc.ForEachPath(func(dir string) error {
				needsFormat, diffOutput, err := formatCheck(ctx, configPath, dir)
				if err != nil {
					return err
				}
				if !needsFormat {
					pocket.Println(ctx, "No files in need of formatting.")
					return nil
				}

				// Show diff in verbose mode.
				if rc.Verbose && len(diffOutput) > 0 {
					pocket.Printf(ctx, "%s", diffOutput)
				}

				// Now actually format.
				if err := ruff.Run(ctx, "format", "--config", configPath, dir); err != nil {
					return fmt.Errorf("ruff format failed in %s: %w", dir, err)
				}
				pocket.Println(ctx, "Formatted files.")
				return nil
			})
		},
	}
}

// LintOptions configures the py-lint task.
type LintOptions struct {
	RuffConfig string `usage:"path to ruff config file"`
}

// LintTask returns a task that lints Python files using ruff check.
// Optional defaults can be passed to set project-level configuration.
func LintTask(defaults ...LintOptions) *pocket.Task {
	return &pocket.Task{
		Name:    "py-lint",
		Usage:   "lint Python files",
		Options: pocket.FirstOrZero(defaults...),
		Action: func(ctx context.Context, rc *pocket.RunContext) error {
			opts := pocket.GetOptions[LintOptions](rc)
			configPath := opts.RuffConfig
			if configPath == "" {
				var err error
				configPath, err = ruff.ConfigPath()
				if err != nil {
					return fmt.Errorf("get ruff config: %w", err)
				}
			}
			return rc.ForEachPath(func(dir string) error {
				if err := ruff.Run(ctx, "check", "--config", configPath, dir); err != nil {
					return fmt.Errorf("ruff check failed in %s: %w", dir, err)
				}
				return nil
			})
		},
	}
}

// TypecheckTask returns a task that type-checks Python files using mypy.
func TypecheckTask() *pocket.Task {
	return &pocket.Task{
		Name:  "py-typecheck",
		Usage: "type-check Python files",
		Action: func(ctx context.Context, rc *pocket.RunContext) error {
			return rc.ForEachPath(func(dir string) error {
				if err := mypy.Run(ctx, dir); err != nil {
					return fmt.Errorf("mypy failed in %s: %w", dir, err)
				}
				return nil
			})
		},
	}
}
