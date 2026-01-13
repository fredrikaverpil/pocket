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
	fix    FixOptions
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
// Use pocket.Paths(golang.Tasks()).DetectBy(golang.Detect()) to enable path filtering.
//
// Execution order: format and fix run first, then lint,
// then test and vulncheck run in parallel.
//
// Example with options:
//
//	pocket.Paths(golang.Tasks(
//	    golang.WithFormat(golang.FormatOptions{LintConfig: ".golangci.yml"}),
//	    golang.WithTest(golang.TestOptions{SkipRace: true}),
//	)).DetectBy(golang.Detect())
func Tasks(opts ...TasksOption) pocket.Runnable {
	var cfg tasksConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	format := FormatTask().WithOptions(cfg.format)
	fix := FixTask().WithOptions(cfg.fix)
	lint := LintTask().WithOptions(cfg.lint)
	test := TestTask().WithOptions(cfg.test)
	vulncheck := VulncheckTask()

	return pocket.Serial(format, fix, lint, pocket.Parallel(test, vulncheck))
}

// Detect returns a detection function that finds Go modules.
// It detects directories containing go.mod files.
//
// Usage:
//
//	pocket.Paths(golang.Tasks()).DetectBy(golang.Detect())
func Detect() func() []string {
	return func() []string {
		return pocket.DetectByFile("go.mod")
	}
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
		configPath, err = golangcilint.Tool.ConfigPath()
		if err != nil {
			return fmt.Errorf("get golangci-lint config: %w", err)
		}
	}

	cmd, err := golangcilint.Tool.Command(ctx, tc, "fmt", "-c", configPath, "./...")
	if err != nil {
		return fmt.Errorf("prepare golangci-lint: %w", err)
	}
	cmd.Dir = pocket.FromGitRoot(tc.Path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("golangci-lint fmt failed in %s: %w", tc.Path, err)
	}
	return nil
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
		configPath, err = golangcilint.Tool.ConfigPath()
		if err != nil {
			return fmt.Errorf("get golangci-lint config: %w", err)
		}
	}

	cmd, err := golangcilint.Tool.Command(ctx, tc, "run", "--allow-parallel-runners", "-c", configPath, "./...")
	if err != nil {
		return fmt.Errorf("prepare golangci-lint: %w", err)
	}
	cmd.Dir = pocket.FromGitRoot(tc.Path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("golangci-lint failed in %s: %w", tc.Path, err)
	}
	return nil
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
		if tc.Path != "." {
			// Replace path separators with dashes for valid filename.
			coverName = "coverage-" + strings.ReplaceAll(tc.Path, "/", "-") + ".out"
		}
		coverFile := pocket.FromGitRoot(coverName)
		args = append(args, "-coverprofile="+coverFile)
	}
	args = append(args, "./...")

	cmd := tc.Command(ctx, "go", args...)
	cmd.Dir = pocket.FromGitRoot(tc.Path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go test failed in %s: %w", tc.Path, err)
	}
	return nil
}

// VulncheckTask returns a task that runs govulncheck.
func VulncheckTask() *pocket.Task {
	return pocket.NewTask("go-vulncheck", "run govulncheck", vulncheckAction)
}

// vulncheckAction is the action for the go-vulncheck task.
func vulncheckAction(ctx context.Context, tc *pocket.TaskContext) error {
	cmd, err := govulncheck.Tool.Command(ctx, tc, "./...")
	if err != nil {
		return fmt.Errorf("prepare govulncheck: %w", err)
	}
	cmd.Dir = pocket.FromGitRoot(tc.Path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("govulncheck failed in %s: %w", tc.Path, err)
	}
	return nil
}

// FixOptions configures the go-fix task.
type FixOptions struct {
	Fixes string `usage:"comma-separated list of fixes to run (default: all)"`
}

// FixTask returns a task that runs go fix to update packages to new APIs.
// Use WithOptions to set project-level configuration.
func FixTask() *pocket.Task {
	return pocket.NewTask("go-fix", "update packages to use new APIs", fixAction)
}

// fixAction is the action for the go-fix task.
func fixAction(ctx context.Context, tc *pocket.TaskContext) error {
	opts := pocket.GetOptions[FixOptions](tc)

	args := []string{"fix"}
	if opts.Fixes != "" {
		args = append(args, "-fix", opts.Fixes)
	}
	args = append(args, "./...")

	cmd := tc.Command(ctx, "go", args...)
	cmd.Dir = pocket.FromGitRoot(tc.Path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go fix failed in %s: %w", tc.Path, err)
	}
	return nil
}
