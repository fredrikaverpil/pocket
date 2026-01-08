// Package golang provides Go-related build tasks.
package golang

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
	"github.com/fredrikaverpil/pocket/tools/govulncheck"
)

// Tasks returns a Runnable that executes all Go tasks in order.
// Tasks auto-detect Go modules by finding go.mod files.
func Tasks() pocket.Runnable {
	return pocket.Serial(
		FormatTask(),
		LintTask(),
		TestTask(),
		VulncheckTask(),
	)
}

// detectModules returns directories containing go.mod files.
func detectModules() []string {
	return pocket.DetectByFile("go.mod")
}

// FormatTask returns a task that formats Go code using golangci-lint fmt.
func FormatTask() *pocket.Task {
	return &pocket.Task{
		Name:  "go-format",
		Usage: "format Go code (gofumpt, goimports, gci, golines)",
		Action: func(ctx context.Context, _ map[string]string) error {
			configPath, err := golangcilint.ConfigPath()
			if err != nil {
				return fmt.Errorf("get golangci-lint config: %w", err)
			}

			for _, dir := range detectModules() {
				cmd, err := golangcilint.Command(ctx, "fmt", "-c", configPath, "./...")
				if err != nil {
					return fmt.Errorf("prepare golangci-lint: %w", err)
				}
				cmd.Dir = pocket.FromGitRoot(dir)
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("golangci-lint fmt failed in %s: %w", dir, err)
				}
			}
			return nil
		},
	}
}

// LintTask returns a task that runs golangci-lint.
func LintTask() *pocket.Task {
	return &pocket.Task{
		Name:  "go-lint",
		Usage: "run golangci-lint",
		Action: func(ctx context.Context, _ map[string]string) error {
			configPath, err := golangcilint.ConfigPath()
			if err != nil {
				return fmt.Errorf("get golangci-lint config: %w", err)
			}

			for _, dir := range detectModules() {
				cmd, err := golangcilint.Command(ctx, "run", "--allow-parallel-runners", "-c", configPath, "./...")
				if err != nil {
					return fmt.Errorf("prepare golangci-lint: %w", err)
				}
				cmd.Dir = pocket.FromGitRoot(dir)
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("golangci-lint failed in %s: %w", dir, err)
				}
			}
			return nil
		},
	}
}

// TestTask returns a task that runs Go tests with race detection.
func TestTask() *pocket.Task {
	return &pocket.Task{
		Name:  "go-test",
		Usage: "run Go tests",
		Action: func(ctx context.Context, _ map[string]string) error {
			for _, dir := range detectModules() {
				args := []string{"test"}
				if pocket.IsVerbose(ctx) {
					args = append(args, "-v")
				}
				args = append(args, "-race", "./...")

				cmd := pocket.Command(ctx, "go", args...)
				cmd.Dir = pocket.FromGitRoot(dir)
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("go test failed in %s: %w", dir, err)
				}
			}
			return nil
		},
	}
}

// VulncheckTask returns a task that runs govulncheck.
func VulncheckTask() *pocket.Task {
	return &pocket.Task{
		Name:  "go-vulncheck",
		Usage: "run govulncheck",
		Action: func(ctx context.Context, _ map[string]string) error {
			for _, dir := range detectModules() {
				cmd, err := govulncheck.Command(ctx, "./...")
				if err != nil {
					return fmt.Errorf("prepare govulncheck: %w", err)
				}
				cmd.Dir = pocket.FromGitRoot(dir)
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("govulncheck failed in %s: %w", dir, err)
				}
			}
			return nil
		},
	}
}
