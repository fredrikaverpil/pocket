package python

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/run"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// LintFlags holds flags for the Lint task.
type LintFlags struct {
	Python  string `flag:"python"   usage:"Python version (for target-version inference)"`
	SkipFix bool   `flag:"skip-fix" usage:"don't auto-fix issues"`
}

// Lint lints Python files using ruff check with auto-fix enabled by default.
// Requires ruff as a project dependency in pyproject.toml.
// Python version can be set via the -python flag.
var Lint = &pk.Task{
	Name:  "py-lint",
	Usage: "lint Python files",
	Flags: LintFlags{},
	Body:  pk.Serial(uv.Install, lintSyncCmd(), lintCmd()),
}

func lintSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := run.GetFlags[LintFlags](ctx)
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: f.Python,
			AllGroups:     true,
		})
	})
}

func lintCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := run.GetFlags[LintFlags](ctx)
		return runLint(ctx, f.Python, f.SkipFix)
	})
}

func runLint(ctx context.Context, pythonVersion string, skipFix bool) error {
	args := []string{
		"check",
		"--exclude", ".pocket",
	}
	if run.Verbose(ctx) {
		args = append(args, "--verbose")
	}
	if !skipFix {
		args = append(args, "--fix")
	}
	if pythonVersion != "" {
		args = append(args, "--target-version", pythonVersionToRuff(pythonVersion))
	}
	args = append(args, run.PathFromContext(ctx))

	return uv.Run(ctx, uv.RunOptions{PythonVersion: pythonVersion}, "ruff", args...)
}
