package python

import (
	"context"
	"flag"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/pcontext"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

var (
	lintFlags   = flag.NewFlagSet("py-lint", flag.ContinueOnError)
	lintPyVer   = lintFlags.String("python", "", "Python version (for target-version inference)")
	lintSkipFix = lintFlags.Bool("skip-fix", false, "don't auto-fix issues")
)

// Lint lints Python files using ruff check with auto-fix enabled by default.
// Requires ruff as a project dependency in pyproject.toml.
// Python version can be set via the -python flag.
var Lint = pk.NewTask("py-lint", "lint Python files", lintFlags,
	pk.Serial(uv.Install, lintSyncCmd(), lintCmd()),
)

func lintSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := resolveVersion(ctx, *lintPyVer)
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: version,
			AllGroups:     true,
		})
	})
}

func lintCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := resolveVersion(ctx, *lintPyVer)
		return runLint(ctx, version, *lintSkipFix)
	})
}

func runLint(ctx context.Context, pythonVersion string, skipFix bool) error {
	args := []string{
		"check",
		"--exclude", ".pocket",
	}
	if pcontext.Verbose(ctx) {
		args = append(args, "--verbose")
	}
	if !skipFix {
		args = append(args, "--fix")
	}
	if pythonVersion != "" {
		args = append(args, "--target-version", pythonVersionToRuff(pythonVersion))
	}
	args = append(args, pcontext.PathFromContext(ctx))

	return uv.Run(ctx, uv.RunOptions{PythonVersion: pythonVersion}, "ruff", args...)
}
