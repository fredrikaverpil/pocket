package python

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// FlagTestCoverage is the flag name for enabling coverage in the Test task.
const FlagTestCoverage = "coverage"

// Test runs Python tests using pytest.
// Requires pytest as a project dependency in pyproject.toml.
// Python version can be set via the -python flag.
// Coverage can be enabled via the -coverage flag.
var Test = &pk.Task{
	Name:  "py-test",
	Usage: "run Python tests",
	Flags: map[string]pk.FlagDef{
		FlagTestCoverage: {Default: false, Usage: "enable coverage reporting"},
		FlagPython:       {Default: "", Usage: "Python version to use (e.g., 3.9)"},
	},
	Body: pk.Serial(uv.Install, testSyncCmd(), testCmd()),
}

func testSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := resolveVersion(ctx, pk.GetFlag[string](ctx, FlagPython))
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: version,
			AllGroups:     true,
		})
	})
}

func testCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := resolveVersion(ctx, pk.GetFlag[string](ctx, FlagPython))
		return runTest(ctx, version, !pk.GetFlag[bool](ctx, FlagTestCoverage))
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
