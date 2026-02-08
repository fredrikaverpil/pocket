package python

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// FlagPython is the flag name for specifying the Python version.
// Shared across Lint, Format, Test, and Typecheck tasks.
const FlagPython = "python"

// Format formats Python files using ruff format.
// Requires ruff as a project dependency in pyproject.toml.
// Python version can be set via the -python flag.
var Format = &pk.Task{
	Name:  "py-format",
	Usage: "format Python files",
	Flags: map[string]pk.FlagDef{
		FlagPython: {Default: "", Usage: "Python version (for target-version inference)"},
	},
	Body: pk.Serial(uv.Install, formatSyncCmd(), formatCmd()),
}

func formatSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := resolveVersion(ctx, pk.GetFlag[string](ctx, FlagPython))
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: version,
			AllGroups:     true,
		})
	})
}

func formatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := resolveVersion(ctx, pk.GetFlag[string](ctx, FlagPython))
		return runFormat(ctx, version)
	})
}

func runFormat(ctx context.Context, pythonVersion string) error {
	args := []string{
		"format",
		"--exclude", ".pocket",
	}
	if pk.Verbose(ctx) {
		args = append(args, "--verbose")
	}
	if pythonVersion != "" {
		args = append(args, "--target-version", pythonVersionToRuff(pythonVersion))
	}
	args = append(args, pk.PathFromContext(ctx))

	return uv.Run(ctx, uv.RunOptions{PythonVersion: pythonVersion}, "ruff", args...)
}

// resolveVersion returns the Python version from the flag.
func resolveVersion(_ context.Context, flagValue string) string {
	return flagValue
}
