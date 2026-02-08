package python

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// FlagLintSkipFix is the flag name for skipping auto-fix in the Lint task.
const FlagLintSkipFix = "skip-fix"

// Lint lints Python files using ruff check with auto-fix enabled by default.
// Requires ruff as a project dependency in pyproject.toml.
// Python version can be set via the -python flag.
var Lint = &pk.Task{
	Name:  "py-lint",
	Usage: "lint Python files",
	Flags: map[string]pk.FlagDef{
		FlagPython:      {Default: "", Usage: "Python version (for target-version inference)"},
		FlagLintSkipFix: {Default: false, Usage: "don't auto-fix issues"},
	},
	Body: pk.Serial(uv.Install, lintSyncCmd(), lintCmd()),
}

func lintSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := resolveVersion(ctx, pk.GetFlag[string](ctx, FlagPython))
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: version,
			AllGroups:     true,
		})
	})
}

func lintCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := resolveVersion(ctx, pk.GetFlag[string](ctx, FlagPython))
		return runLint(ctx, version, pk.GetFlag[bool](ctx, FlagLintSkipFix))
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
