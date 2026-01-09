// Package golang provides Go-related build tasks.
package golang

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
	"github.com/fredrikaverpil/pocket/tools/govulncheck"
)

// Options configures the Go tasks.
type Options struct {
	// LintConfig is the path to golangci-lint config file.
	// If empty, uses the default config from pocket.
	LintConfig string

	// TestRace enables race detection in tests.
	// Default: true
	TestRace *bool
}

// testRace returns the effective TestRace value (defaults to true).
func (o Options) testRace() bool {
	if o.TestRace == nil {
		return true
	}
	return *o.TestRace
}

// Tasks returns a Runnable that executes all Go tasks.
// Tasks auto-detect Go modules by finding go.mod files.
// Use pocket.AutoDetect(golang.Tasks()) to enable path filtering.
func Tasks(opts ...Options) pocket.Runnable {
	return &goTasks{
		format:    FormatTask(opts...),
		lint:      LintTask(opts...),
		test:      TestTask(opts...),
		vulncheck: VulncheckTask(opts...),
	}
}

// goTasks is the Runnable for Go tasks that also implements Detectable.
type goTasks struct {
	format    *pocket.Task
	lint      *pocket.Task
	test      *pocket.Task
	vulncheck *pocket.Task
}

// Run executes all Go tasks.
func (g *goTasks) Run(ctx context.Context) error {
	if err := pocket.Serial(g.format, g.lint).Run(ctx); err != nil {
		return err
	}
	return pocket.Parallel(g.test, g.vulncheck).Run(ctx)
}

// Tasks returns all Go tasks.
func (g *goTasks) Tasks() []*pocket.Task {
	return []*pocket.Task{g.format, g.lint, g.test, g.vulncheck}
}

// DefaultDetect returns a function that detects Go module directories.
func (g *goTasks) DefaultDetect() func() []string {
	return detectModules
}

// detectModules returns directories containing go.mod files.
func detectModules() []string {
	return pocket.DetectByFile("go.mod")
}

// FormatTask returns a task that formats Go code using golangci-lint fmt.
func FormatTask(opts ...Options) *pocket.Task {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	return &pocket.Task{
		Name:  "go-format",
		Usage: "format Go code (gofumpt, goimports, gci, golines)",
		Action: func(ctx context.Context, taskOpts *pocket.RunContext) error {
			configPath := o.LintConfig
			if configPath == "" {
				var err error
				configPath, err = golangcilint.ConfigPath()
				if err != nil {
					return fmt.Errorf("get golangci-lint config: %w", err)
				}
			}

			for _, dir := range taskOpts.Paths {
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
func LintTask(opts ...Options) *pocket.Task {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	return &pocket.Task{
		Name:  "go-lint",
		Usage: "run golangci-lint",
		Action: func(ctx context.Context, taskOpts *pocket.RunContext) error {
			configPath := o.LintConfig
			if configPath == "" {
				var err error
				configPath, err = golangcilint.ConfigPath()
				if err != nil {
					return fmt.Errorf("get golangci-lint config: %w", err)
				}
			}

			for _, dir := range taskOpts.Paths {
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
func TestTask(opts ...Options) *pocket.Task {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	return &pocket.Task{
		Name:  "go-test",
		Usage: "run Go tests",
		Action: func(ctx context.Context, taskOpts *pocket.RunContext) error {
			for _, dir := range taskOpts.Paths {
				args := []string{"test"}
				if pocket.IsVerbose(ctx) {
					args = append(args, "-v")
				}
				if o.testRace() {
					args = append(args, "-race")
				}
				args = append(args, "./...")

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
func VulncheckTask(_ ...Options) *pocket.Task {
	return &pocket.Task{
		Name:  "go-vulncheck",
		Usage: "run govulncheck",
		Action: func(ctx context.Context, taskOpts *pocket.RunContext) error {
			for _, dir := range taskOpts.Paths {
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
