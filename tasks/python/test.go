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

// testWith creates a test task for a specific Python version without coverage.
func testWith(pythonVersion string) *pk.Task {
	return pk.NewTask("py-test:"+pythonVersion, "run Python tests", nil,
		pk.Serial(uv.Install, testSyncCmdWith(pythonVersion), testCmdWith(pythonVersion, true)),
	)
}

// testWithCoverage creates a test task for a specific Python version with coverage.
func testWithCoverage(pythonVersion string) *pk.Task {
	return pk.NewTask("py-test:"+pythonVersion, "run Python tests with coverage", nil,
		pk.Serial(uv.Install, testSyncCmdWith(pythonVersion), testCmdWith(pythonVersion, false)),
	)
}

func testSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: *testPyVer,
			AllGroups:     true,
		})
	})
}

func testSyncCmdWith(pythonVersion string) pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: pythonVersion,
			AllGroups:     true,
		})
	})
}

func testCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return runTest(ctx, *testPyVer, *testSkipCoverage)
	})
}

func testCmdWith(pythonVersion string, skipCoverage bool) pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return runTest(ctx, pythonVersion, skipCoverage)
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
