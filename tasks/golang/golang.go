// Package golang provides Go development tasks.
// This is a "task" package - it orchestrates tools to do work.
package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
	"github.com/fredrikaverpil/pocket/tools/govulncheck"
)

// Task definitions - these are the visible functions in CLI.
var (
	// Format formats Go code using go fmt.
	Format = pocket.Func("go-format", "format Go code", format)

	// Lint runs golangci-lint.
	Lint = pocket.Func("go-lint", "run golangci-lint", lint).
		With(LintOptions{})

	// Test runs tests with race detection and coverage.
	Test = pocket.Func("go-test", "run Go tests", test).
		With(TestOptions{Race: true, Coverage: true})

	// Vulncheck runs govulncheck for vulnerability scanning.
	Vulncheck = pocket.Func("go-vulncheck", "run govulncheck", vulncheck)
)

// Option configures the golang task group.
type Option func(*config)

type config struct {
	lint LintOptions
	test TestOptions
}

// WithLint sets options for the go-lint task.
func WithLint(opts LintOptions) Option {
	return func(c *config) { c.lint = opts }
}

// WithTest sets options for the go-test task.
func WithTest(opts TestOptions) Option {
	return func(c *config) { c.test = opts }
}

// Workflow returns all Go tasks composed as a Runnable.
// Use this with pocket.Paths().DetectBy() for auto-detection.
//
// Example:
//
//	pocket.Paths(golang.Workflow()).DetectBy(golang.Detect())
//
// Example with options:
//
//	pocket.Paths(golang.Workflow(
//	    golang.WithLint(golang.LintOptions{Config: ".golangci.yml"}),
//	    golang.WithTest(golang.TestOptions{Race: false}),
//	)).DetectBy(golang.Detect())
func Workflow(opts ...Option) pocket.Runnable {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	// Apply options to tasks
	lintTask := Lint
	if cfg.lint != (LintOptions{}) {
		lintTask = Lint.With(cfg.lint)
	}

	testTask := Test
	if cfg.test != (TestOptions{}) {
		testTask = Test.With(cfg.test)
	}

	return pocket.Serial(
		Format,
		lintTask,
		pocket.Parallel(testTask, Vulncheck),
	)
}

// Detect returns a detection function for Go modules.
// It finds directories containing go.mod files.
func Detect() func() []string {
	return func() []string {
		return pocket.DetectByFile("go.mod")
	}
}

// LintOptions configures the go-lint task.
type LintOptions struct {
	Config string `arg:"config" usage:"path to golangci-lint config file"`
	Fix    bool   `arg:"fix"    usage:"auto-fix issues"`
}

// TestOptions configures the go-test task.
type TestOptions struct {
	Race     bool `arg:"race"     usage:"enable race detection"`
	Coverage bool `arg:"coverage" usage:"generate coverage.out in git root"`
	Short    bool `arg:"short"    usage:"run short tests only"`
	Verbose  bool `arg:"verbose"  usage:"verbose output"`
}

// Task implementations.

func format(ctx context.Context) error {
	return pocket.Exec(ctx, "go", "fmt", "./...")
}

func lint(ctx context.Context) error {
	opts := pocket.Options[LintOptions](ctx)

	args := []string{"run"}
	if opts.Config != "" {
		args = append(args, "-c", opts.Config)
	} else if configPath, err := pocket.ConfigPath("golangci-lint", golangcilint.Config); err == nil && configPath != "" {
		args = append(args, "-c", configPath)
	}
	if opts.Fix {
		args = append(args, "--fix")
	}
	args = append(args, "./...")

	return golangcilint.Exec(ctx, args...)
}

func test(ctx context.Context) error {
	opts := pocket.Options[TestOptions](ctx)

	args := []string{"test"}
	if opts.Verbose {
		args = append(args, "-v")
	}
	if opts.Race {
		args = append(args, "-race")
	}
	if opts.Coverage {
		coverPath := pocket.FromGitRoot("coverage.out")
		args = append(args, "-coverprofile="+coverPath)
	}
	if opts.Short {
		args = append(args, "-short")
	}
	args = append(args, "./...")

	return pocket.Exec(ctx, "go", args...)
}

func vulncheck(ctx context.Context) error {
	return govulncheck.Exec(ctx, "./...")
}
