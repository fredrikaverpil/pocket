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

// Test runs Python tests using pytest.
// Requires pytest as a project dependency in pyproject.toml.
// Python version can be set via flag (-python) or via python.WithVersion() option.
// Coverage can be enabled via python.WithCoverage() option.
var Test = pk.NewTask("py-test", "run Python tests", testFlags,
	pk.Serial(uv.Install, testSyncCmd(), testCmd()),
)

func testSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := resolveVersion(ctx, *testPyVer)
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: version,
			AllGroups:     true,
		})
	})
}

func testCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := resolveVersion(ctx, *testPyVer)
		// Coverage enabled via flag (inverted) or context option
		skipCoverage := *testSkipCoverage || !coverageFromContext(ctx)
		return runTest(ctx, version, skipCoverage)
	})
}

func runTest(ctx context.Context, pythonVersion string, skipCoverage bool) error {
	opts := uv.RunOptions{PythonVersion: pythonVersion}

	if skipCoverage {
		args := []string{}
		if pk.Verbose(ctx) {
			args = append(args, "-vv")
		}
		return uv.Run(ctx, opts, "pytest", args...)
	}

	// Run with coverage.
	args := []string{"run", "--parallel-mode", "-m", "pytest"}
	if pk.Verbose(ctx) {
		args = append(args, "-vv")
	}
	if err := uv.Run(ctx, opts, "coverage", args...); err != nil {
		return err
	}

	// Combine parallel coverage files.
	if err := uv.Run(ctx, opts, "coverage", "combine"); err != nil {
		pk.Printf(ctx, "Note: coverage combine skipped (may be single run)\n")
	}

	// Show coverage report.
	if err := uv.Run(ctx, opts, "coverage", "report"); err != nil {
		return err
	}

	// Generate HTML report.
	return uv.Run(ctx, opts, "coverage", "html")
}
