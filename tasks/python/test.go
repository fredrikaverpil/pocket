package python

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// TestOptions configures the py-test task.
type TestOptions struct {
	SkipCoverage bool `arg:"skip-coverage" usage:"disable coverage generation"`
	Verbose      bool `arg:"verbose"       usage:"verbose output (-vv)"`
}

// Test runs Python tests using pytest with coverage by default.
// Requires pytest and coverage as project dependencies in pyproject.toml.
var Test = pocket.Func("py-test", "run Python tests", pocket.Serial(
	uv.Install,
	sync,
	test,
)).With(TestOptions{})

// sync installs project dependencies.
func sync(ctx context.Context) error {
	return pocket.Exec(ctx, uv.Name, "sync", "--all-groups")
}

func test(ctx context.Context) error {
	opts := pocket.Options[TestOptions](ctx)

	if opts.SkipCoverage {
		// Run pytest directly without coverage
		args := []string{"run", "pytest"}
		if opts.Verbose {
			args = append(args, "-vv")
		}
		return pocket.Exec(ctx, uv.Name, args...)
	}

	// Run with coverage: coverage run -m pytest
	args := []string{"run", "coverage", "run", "-m", "pytest"}
	if opts.Verbose {
		args = append(args, "-vv")
	}
	if err := pocket.Exec(ctx, uv.Name, args...); err != nil {
		return err
	}

	// Show coverage report
	if err := pocket.Exec(ctx, uv.Name, "run", "coverage", "report"); err != nil {
		return err
	}

	// Generate HTML report
	return pocket.Exec(ctx, uv.Name, "run", "coverage", "html")
}
