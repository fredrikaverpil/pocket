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

func lintSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return uv.Sync(ctx, *lintPyVer, true)
	})
}

func lintCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		args := []string{
			"check",
			"--exclude", ".pocket",
		}
		if pk.Verbose(ctx) {
			args = append(args, "--verbose")
		}
		if !*lintSkipFix {
			args = append(args, "--fix")
		}
		if *lintPyVer != "" {
			args = append(args, "--target-version", pythonVersionToRuff(*lintPyVer))
		}
		args = append(args, pk.PathFromContext(ctx))

		return uv.Run(ctx, *lintPyVer, "ruff", args...)
	})
}
