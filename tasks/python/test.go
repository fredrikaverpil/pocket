package python

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// TestOptions configures the py-test task.
type TestOptions struct {
	PythonVersion string `arg:"python"        usage:"Python version to use (e.g., 3.9)"`
	SkipCoverage  bool   `arg:"skip-coverage" usage:"disable coverage generation"`
}

// Test runs Python tests using pytest with coverage by default.
// Requires pytest and coverage as project dependencies in pyproject.toml.
var Test = pocket.Task("py-test", "run Python tests",
	pocket.Serial(uv.Install, testSyncCmd(), testCmd()),
	pocket.Opts(TestOptions{}),
)

func testSyncCmd() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		opts := pocket.Options[TestOptions](ctx)
		args := []string{"sync", "--all-groups"}
		if opts.PythonVersion != "" {
			args = append(args, "--python", opts.PythonVersion)
		}
		return pocket.Exec(ctx, uv.Name, args...)
	})
}

func testCmd() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		opts := pocket.Options[TestOptions](ctx)

		// Build base args with optional Python version
		baseArgs := []string{"run"}
		if opts.PythonVersion != "" {
			baseArgs = append(baseArgs, "--python", opts.PythonVersion)
		}

		if opts.SkipCoverage {
			// Run pytest directly without coverage
			args := append(baseArgs, "pytest")
			if pocket.Verbose(ctx) {
				args = append(args, "-vv")
			}
			return pocket.Exec(ctx, uv.Name, args...)
		}

		// Run with coverage: coverage run -m pytest
		args := append(baseArgs, "coverage", "run", "-m", "pytest")
		if pocket.Verbose(ctx) {
			args = append(args, "-vv")
		}
		if err := pocket.Exec(ctx, uv.Name, args...); err != nil {
			return err
		}

		// Show coverage report
		reportArgs := append(baseArgs, "coverage", "report")
		if err := pocket.Exec(ctx, uv.Name, reportArgs...); err != nil {
			return err
		}

		// Generate HTML report
		htmlArgs := append(baseArgs, "coverage", "html")
		return pocket.Exec(ctx, uv.Name, htmlArgs...)
	})
}
