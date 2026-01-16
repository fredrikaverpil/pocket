package python

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// TestOptions configures the py-test task.
type TestOptions struct {
	SkipCoverage bool `arg:"skip-coverage" usage:"disable coverage generation"`
}

// Test runs Python tests using pytest with coverage by default.
// Requires pytest and coverage as project dependencies in pyproject.toml.
var Test = pocket.Task("py-test", "run Python tests",
	pocket.Serial(uv.Install, pocket.Run(uv.Name, "sync", "--all-groups"), testCmd()),
	pocket.Opts(TestOptions{}),
)

func testCmd() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		opts := pocket.Options[TestOptions](ctx)

		if opts.SkipCoverage {
			// Run pytest directly without coverage
			args := []string{"run", "pytest"}
			if pocket.Verbose(ctx) {
				args = append(args, "-vv")
			}
			return pocket.Exec(ctx, uv.Name, args...)
		}

		// Run with coverage: coverage run -m pytest
		args := []string{"run", "coverage", "run", "-m", "pytest"}
		if pocket.Verbose(ctx) {
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
	})
}
