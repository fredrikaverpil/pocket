package python

import (
	"context"
	"flag"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

var (
	lintFlags   = flag.NewFlagSet("py-lint", flag.ContinueOnError)
	lintPyVer   = lintFlags.String("python", "", "Python version (for target-version inference)")
	lintSkipFix = lintFlags.Bool("skip-fix", false, "don't auto-fix issues")
)

// Lint lints Python files using ruff check with auto-fix enabled by default.
// Requires ruff as a project dependency in pyproject.toml.
var Lint = pk.NewTask("py-lint", "lint Python files", lintFlags,
	pk.Serial(uv.Install, lintSyncCmd(), lintCmd()),
)

// lintWith creates a lint task for a specific Python version.
func lintWith(pythonVersion string) *pk.Task {
	return pk.NewTask("py-lint:"+pythonVersion, "lint Python files", nil,
		pk.Serial(uv.Install, lintSyncCmdWith(pythonVersion), lintCmdWith(pythonVersion, false)),
	)
}

func lintSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: *lintPyVer,
			AllGroups:     true,
		})
	})
}

func lintSyncCmdWith(pythonVersion string) pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: pythonVersion,
			AllGroups:     true,
		})
	})
}

func lintCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return runLint(ctx, *lintPyVer, *lintSkipFix)
	})
}

func lintCmdWith(pythonVersion string, skipFix bool) pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return runLint(ctx, pythonVersion, skipFix)
	})
}

func runLint(ctx context.Context, pythonVersion string, skipFix bool) error {
	args := []string{
		"check",
		"--exclude", ".pocket",
	}
	if pk.Verbose(ctx) {
		args = append(args, "--verbose")
	}
	if !skipFix {
		args = append(args, "--fix")
	}
	if pythonVersion != "" {
		args = append(args, "--target-version", pythonVersionToRuff(pythonVersion))
	}
	args = append(args, pk.PathFromContext(ctx))

	return uv.Run(ctx, uv.RunOptions{PythonVersion: pythonVersion}, "ruff", args...)
}
