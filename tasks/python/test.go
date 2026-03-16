package python

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/run"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// TestFlags holds flags for the Test task.
type TestFlags struct {
	Coverage bool   `flag:"coverage" usage:"enable coverage reporting"`
	Python   string `flag:"python"   usage:"Python version to use (e.g., 3.9)"`
}

// Test runs Python tests using pytest.
// Requires pytest as a project dependency in pyproject.toml.
// Python version can be set via the -python flag.
// Coverage can be enabled via the -coverage flag.
var Test = &pk.Task{
	Name:  "py-test",
	Usage: "run Python tests",
	Flags: TestFlags{},
	Body:  pk.Serial(uv.Install, testSyncCmd(), testCmd()),
}

func testSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := run.GetFlags[TestFlags](ctx)
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: f.Python,
			AllGroups:     true,
		})
	})
}

func testCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := run.GetFlags[TestFlags](ctx)
		return runTest(ctx, f.Python, !f.Coverage)
	})
}

func runTest(ctx context.Context, pythonVersion string, skipCoverage bool) error {
	opts := uv.RunOptions{PythonVersion: pythonVersion}

	if skipCoverage {
		args := []string{}
		if run.Verbose(ctx) {
			args = append(args, "-vv")
		}
		return uv.Run(ctx, opts, "pytest", args...)
	}

	// Run with coverage.
	args := []string{"run", "--parallel-mode", "-m", "pytest"}
	if run.Verbose(ctx) {
		args = append(args, "-vv")
	}
	if err := uv.Run(ctx, opts, "coverage", args...); err != nil {
		return err
	}

	// Combine parallel coverage files.
	if err := uv.Run(ctx, opts, "coverage", "combine"); err != nil {
		run.Printf(ctx, "Note: coverage combine skipped (may be single run)\n")
	}

	// Show coverage report.
	if err := uv.Run(ctx, opts, "coverage", "report"); err != nil {
		return err
	}

	// Generate HTML report.
	return uv.Run(ctx, opts, "coverage", "html")
}
