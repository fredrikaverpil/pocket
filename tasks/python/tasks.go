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
// Tasks auto-detect Python projects by finding pyproject.toml, setup.py, or setup.cfg.
// Use pocket.AutoDetect(python.Tasks()) to enable path filtering.
//
// Execution order: format runs first, then lint and typecheck run in parallel.
//
// Example with options:
//
//	pocket.AutoDetect(python.Tasks(
//	    python.WithFormat(python.FormatOptions{RuffConfig: "ruff.toml"}),
//	))
func Tasks(opts ...TasksOption) pocket.Runnable {
	var cfg tasksConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	format := FormatTask().WithOptions(cfg.format)
	lint := LintTask().WithOptions(cfg.lint)
	typecheck := TypecheckTask()

	return pocket.NewTaskGroup(format, lint, typecheck).
		RunWith(func(ctx context.Context, exec *pocket.Execution) error {
			// Format must run first.
			if err := format.Run(ctx, exec); err != nil {
				return err
			}
			// Lint and typecheck can run in parallel.
			return pocket.Parallel(lint, typecheck).Run(ctx, exec)
		}).
		DetectByFile("pyproject.toml", "setup.py", "setup.cfg")
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
		configPath, err = ruff.ConfigPath()
		if err != nil {
			return fmt.Errorf("get ruff config: %w", err)
		}
	}
	return tc.ForEachPath(ctx, func(dir string) error {
		needsFormat, diffOutput, err := formatCheck(ctx, configPath, dir)
		if err != nil {
			return err
		}
		if !needsFormat {
			tc.Out.Println("No files in need of formatting.")
			return nil
		}

		// Show diff in verbose mode.
		if tc.Verbose && len(diffOutput) > 0 {
			tc.Out.Printf("%s", diffOutput)
		}

		// Now actually format.
		if err := ruff.Run(ctx, "format", "--config", configPath, dir); err != nil {
			return fmt.Errorf("ruff format failed in %s: %w", dir, err)
		}
		tc.Out.Println("Formatted files.")
		return nil
	})
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
		configPath, err = ruff.ConfigPath()
		if err != nil {
			return fmt.Errorf("get ruff config: %w", err)
		}
	}
	return tc.ForEachPath(ctx, func(dir string) error {
		if err := ruff.Run(ctx, "check", "--config", configPath, dir); err != nil {
			return fmt.Errorf("ruff check failed in %s: %w", dir, err)
		}
		return nil
	})
}

// TypecheckTask returns a task that type-checks Python files using mypy.
func TypecheckTask() *pocket.Task {
	return pocket.NewTask("py-typecheck", "type-check Python files", typecheckAction)
}

// typecheckAction is the action for the py-typecheck task.
func typecheckAction(ctx context.Context, tc *pocket.TaskContext) error {
	return tc.ForEachPath(ctx, func(dir string) error {
		if err := mypy.Run(ctx, dir); err != nil {
			return fmt.Errorf("mypy failed in %s: %w", dir, err)
		}
		return nil
	})
}
