// Package golang provides Go-related build tasks.
package golang

import (
	"context"
	"fmt"
	"strings"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
	"github.com/fredrikaverpil/pocket/tools/govulncheck"
)

// TasksOption configures the golang task group.
type TasksOption func(*tasksConfig)

type tasksConfig struct {
	format FormatOptions
	lint   LintOptions
	test   TestOptions
}

// WithFormat sets options for the go-format task.
func WithFormat(opts FormatOptions) TasksOption {
	return func(c *tasksConfig) { c.format = opts }
}

// WithLint sets options for the go-lint task.
func WithLint(opts LintOptions) TasksOption {
	return func(c *tasksConfig) { c.lint = opts }
}

// WithTest sets options for the go-test task.
func WithTest(opts TestOptions) TasksOption {
	return func(c *tasksConfig) { c.test = opts }
}

// Tasks returns a Runnable that executes all Go tasks.
// Tasks auto-detect Go modules by finding go.mod files.
// Use pocket.AutoDetect(golang.Tasks()) to enable path filtering.
//
// Execution order: format and lint run serially first,
// then test and vulncheck run in parallel.
//
// Example with options:
//
//	pocket.AutoDetect(golang.Tasks(
//	    golang.WithFormat(golang.FormatOptions{LintConfig: ".golangci.yml"}),
//	    golang.WithTest(golang.TestOptions{SkipRace: true}),
//	))
func Tasks(opts ...TasksOption) pocket.Runnable {
	var cfg tasksConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	format := FormatTask().WithOptions(cfg.format)
	lint := LintTask().WithOptions(cfg.lint)
	test := TestTask().WithOptions(cfg.test)
	vulncheck := VulncheckTask()

	return pocket.NewTaskGroup(format, lint, test, vulncheck).
		RunWith(func(ctx context.Context, exec *pocket.Execution) error {
			// Format and lint must run serially (lint after format).
			if err := pocket.Serial(format, lint).Run(ctx, exec); err != nil {
				return err
			}
			// Test and vulncheck can run in parallel.
			return pocket.Parallel(test, vulncheck).Run(ctx, exec)
		}).
		DetectByFile("go.mod")
}

// FormatOptions configures the go-format task.
type FormatOptions struct {
	LintConfig string `usage:"path to golangci-lint config file"`
}

// FormatTask returns a task that formats Go code using golangci-lint fmt.
// Use WithOptions to set project-level configuration.
func FormatTask() *pocket.Task {
	return pocket.NewTask("go-format", "format Go code (gofumpt, goimports, gci, golines)", formatAction)
}

// formatAction is the action for the go-format task.
func formatAction(ctx context.Context, tc *pocket.TaskContext) error {
	opts := pocket.GetOptions[FormatOptions](tc)
	configPath := opts.LintConfig
	if configPath == "" {
		var err error
		configPath, err = golangcilint.ConfigPath()
		if err != nil {
			return fmt.Errorf("get golangci-lint config: %w", err)
		}
	}
	return tc.ForEachPath(ctx, func(dir string) error {
		absDir := pocket.FromGitRoot(dir)

		needsFormat, diffOutput, err := formatCheck(ctx, configPath, absDir)
		if err != nil {
			return err
		}
		if !needsFormat {
			tc.Out.Println("No files in need of formatting.")
			return nil
		}

		// Show diff in verbose mode.
		if tc.Verbose {
			tc.Out.Printf("%s", diffOutput)
		}

		// Now actually format.
		cmd, err := golangcilint.Command(ctx, "fmt", "-c", configPath, "./...")
		if err != nil {
			return fmt.Errorf("prepare golangci-lint: %w", err)
		}
		cmd.Dir = absDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("golangci-lint fmt failed in %s: %w", dir, err)
		}
		tc.Out.Println("Formatted files.")
		return nil
	})
}

// formatCheck runs golangci-lint fmt --diff to check if formatting is needed.
// Returns true if files need formatting, along with the diff output.
func formatCheck(ctx context.Context, configPath, dir string) (needsFormat bool, output []byte, err error) {
	cmd, err := golangcilint.Command(ctx, "fmt", "-c", configPath, "--diff", "./...")
	if err != nil {
		return false, nil, fmt.Errorf("prepare golangci-lint: %w", err)
	}
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	output, _ = cmd.CombinedOutput()
	return len(output) > 0, output, nil
}

// LintOptions configures the go-lint task.
type LintOptions struct {
	LintConfig string `usage:"path to golangci-lint config file"`
}

// LintTask returns a task that runs golangci-lint.
// Use WithOptions to set project-level configuration.
func LintTask() *pocket.Task {
	return pocket.NewTask("go-lint", "run golangci-lint", lintAction)
}

// lintAction is the action for the go-lint task.
func lintAction(ctx context.Context, tc *pocket.TaskContext) error {
	opts := pocket.GetOptions[LintOptions](tc)
	configPath := opts.LintConfig
	if configPath == "" {
		var err error
		configPath, err = golangcilint.ConfigPath()
		if err != nil {
			return fmt.Errorf("get golangci-lint config: %w", err)
		}
	}
	return tc.ForEachPath(ctx, func(dir string) error {
		cmd, err := golangcilint.Command(ctx, "run", "--allow-parallel-runners", "-c", configPath, "./...")
		if err != nil {
			return fmt.Errorf("prepare golangci-lint: %w", err)
		}
		cmd.Dir = pocket.FromGitRoot(dir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("golangci-lint failed in %s: %w", dir, err)
		}
		return nil
	})
}

// TestOptions configures the go-test task.
type TestOptions struct {
	SkipRace     bool `usage:"skip race detection"`
	SkipCoverage bool `usage:"skip coverage output"`
}

// TestTask returns a task that runs Go tests with race detection and coverage.
// Use WithOptions to set project-level configuration.
func TestTask() *pocket.Task {
	return pocket.NewTask("go-test", "run Go tests", testAction)
}

// testAction is the action for the go-test task.
func testAction(ctx context.Context, tc *pocket.TaskContext) error {
	opts := pocket.GetOptions[TestOptions](tc)
	return tc.ForEachPath(ctx, func(dir string) error {
		args := []string{"test"}
		if tc.Verbose {
			args = append(args, "-v")
		}
		if !opts.SkipRace {
			args = append(args, "-race")
		}
		if !opts.SkipCoverage {
			// Name coverage file based on directory to avoid overwrites.
			coverName := "coverage.out"
			if dir != "." {
				// Replace path separators with dashes for valid filename.
				coverName = "coverage-" + strings.ReplaceAll(dir, "/", "-") + ".out"
			}
			coverFile := pocket.FromGitRoot(coverName)
			args = append(args, "-coverprofile="+coverFile)
		}
		args = append(args, "./...")

		cmd := pocket.Command(ctx, "go", args...)
		cmd.Dir = pocket.FromGitRoot(dir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go test failed in %s: %w", dir, err)
		}
		return nil
	})
}

// VulncheckTask returns a task that runs govulncheck.
func VulncheckTask() *pocket.Task {
	return pocket.NewTask("go-vulncheck", "run govulncheck", vulncheckAction)
}

// vulncheckAction is the action for the go-vulncheck task.
func vulncheckAction(ctx context.Context, tc *pocket.TaskContext) error {
	return tc.ForEachPath(ctx, func(dir string) error {
		cmd, err := govulncheck.Command(ctx, "./...")
		if err != nil {
			return fmt.Errorf("prepare govulncheck: %w", err)
		}
		cmd.Dir = pocket.FromGitRoot(dir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("govulncheck failed in %s: %w", dir, err)
		}
		return nil
	})
}
