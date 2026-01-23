package python

import (
	"context"
	"flag"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

var (
	testFlags        = flag.NewFlagSet("py-test", flag.ContinueOnError)
	testPyVer        = testFlags.String("python", "", "Python version to use (e.g., 3.9)")
	testSkipCoverage = testFlags.Bool("skip-coverage", false, "disable coverage generation")
)

// Test runs Python tests using pytest with coverage by default.
// Requires pytest and coverage as project dependencies in pyproject.toml.
var Test = pk.NewTask("py-test", "run Python tests", testFlags,
	pk.Serial(uv.Install, testSyncCmd(), testCmd()),
)

func testSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return uv.Sync(ctx, *testPyVer, true)
	})
}

func testCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		if *testSkipCoverage {
			args := []string{}
			if pk.Verbose(ctx) {
				args = append(args, "-vv")
			}
			return uv.Run(ctx, *testPyVer, "pytest", args...)
		}

		// Run with coverage.
		args := []string{"run", "--parallel-mode", "-m", "pytest"}
		if pk.Verbose(ctx) {
			args = append(args, "-vv")
		}
		if err := uv.Run(ctx, *testPyVer, "coverage", args...); err != nil {
			return err
		}

		// Combine parallel coverage files.
		if err := uv.Run(ctx, *testPyVer, "coverage", "combine"); err != nil {
			pk.Printf(ctx, "Note: coverage combine skipped (may be single run)\n")
		}

		// Show coverage report.
		if err := uv.Run(ctx, *testPyVer, "coverage", "report"); err != nil {
			return err
		}

		// Generate HTML report.
		return uv.Run(ctx, *testPyVer, "coverage", "html")
	})
}
