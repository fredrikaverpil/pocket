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

// Tasks returns a Runnable that executes all Go tasks.
// Tasks auto-detect Go modules by finding go.mod files.
// Use pocket.AutoDetect(golang.Tasks()) to enable path filtering.
//
// Execution order: format and lint run serially first,
// then test and vulncheck run in parallel.
func Tasks() pocket.Runnable {
	format := FormatTask()
	lint := LintTask()
	test := TestTask()
	vulncheck := VulncheckTask()

	return pocket.NewTaskGroup(format, lint, test, vulncheck).
		RunWith(func(ctx context.Context) error {
			// Format and lint must run serially (lint after format).
			if err := pocket.Serial(format, lint).Run(ctx); err != nil {
				return err
			}
			// Test and vulncheck can run in parallel.
			return pocket.Parallel(test, vulncheck).Run(ctx)
		}).
		DetectByFile("go.mod")
}

// FormatOptions configures the go-format task.
type FormatOptions struct {
	LintConfig string `usage:"path to golangci-lint config file"`
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

// FormatTask returns a task that formats Go code using golangci-lint fmt.
// Optional defaults can be passed to set project-level configuration.
func FormatTask(defaults ...FormatOptions) *pocket.Task {
	return &pocket.Task{
		Name:    "go-format",
		Usage:   "format Go code (gofumpt, goimports, gci, golines)",
		Options: pocket.FirstOrZero(defaults...),
		Action: func(ctx context.Context, rc *pocket.RunContext) error {
			opts := pocket.GetOptions[FormatOptions](rc)
			configPath := opts.LintConfig
			if configPath == "" {
				var err error
				configPath, err = golangcilint.ConfigPath()
				if err != nil {
					return fmt.Errorf("get golangci-lint config: %w", err)
				}
			}
			return rc.ForEachPath(func(dir string) error {
				absDir := pocket.FromGitRoot(dir)

				needsFormat, diffOutput, err := formatCheck(ctx, configPath, absDir)
				if err != nil {
					return err
				}
				if !needsFormat {
					pocket.Println(ctx, "No files in need of formatting.")
					return nil
				}

				// Show diff in verbose mode.
				if rc.Verbose {
					pocket.Printf(ctx, "%s", diffOutput)
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
				pocket.Println(ctx, "Formatted files.")
				return nil
			})
		},
	}
}

// LintOptions configures the go-lint task.
type LintOptions struct {
	LintConfig string `usage:"path to golangci-lint config file"`
}

// LintTask returns a task that runs golangci-lint.
// Optional defaults can be passed to set project-level configuration.
func LintTask(defaults ...LintOptions) *pocket.Task {
	return &pocket.Task{
		Name:    "go-lint",
		Usage:   "run golangci-lint",
		Options: pocket.FirstOrZero(defaults...),
		Action: func(ctx context.Context, rc *pocket.RunContext) error {
			opts := pocket.GetOptions[LintOptions](rc)
			configPath := opts.LintConfig
			if configPath == "" {
				var err error
				configPath, err = golangcilint.ConfigPath()
				if err != nil {
					return fmt.Errorf("get golangci-lint config: %w", err)
				}
			}
			return rc.ForEachPath(func(dir string) error {
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
		},
	}
}

// TestOptions configures the go-test task.
type TestOptions struct {
	SkipRace     bool `usage:"skip race detection"`
	SkipCoverage bool `usage:"skip coverage output"`
}

// TestTask returns a task that runs Go tests with race detection and coverage.
// Optional defaults can be passed to set project-level configuration.
func TestTask(defaults ...TestOptions) *pocket.Task {
	return &pocket.Task{
		Name:    "go-test",
		Usage:   "run Go tests",
		Options: pocket.FirstOrZero(defaults...),
		Action: func(ctx context.Context, rc *pocket.RunContext) error {
			opts := pocket.GetOptions[TestOptions](rc)
			return rc.ForEachPath(func(dir string) error {
				args := []string{"test"}
				if rc.Verbose {
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
		},
	}
}

// VulncheckTask returns a task that runs govulncheck.
func VulncheckTask() *pocket.Task {
	return &pocket.Task{
		Name:  "go-vulncheck",
		Usage: "run govulncheck",
		Action: func(ctx context.Context, rc *pocket.RunContext) error {
			return rc.ForEachPath(func(dir string) error {
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
		},
	}
}
