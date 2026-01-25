package python

import (
	"context"
	"flag"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

var (
	formatFlags = flag.NewFlagSet("py-format", flag.ContinueOnError)
	formatPyVer = formatFlags.String("python", "", "Python version (for target-version inference)")
)

// Format formats Python files using ruff format.
// Requires ruff as a project dependency in pyproject.toml.
// Python version can be set via flag (-python) or via python.WithVersion() option.
var Format = pk.NewTask("py-format", "format Python files", formatFlags,
	pk.Serial(uv.Install, formatSyncCmd(), formatCmd()),
)

func formatSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := resolveVersion(ctx, *formatPyVer)
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: version,
			AllGroups:     true,
		})
	})
}

func formatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := resolveVersion(ctx, *formatPyVer)
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

// resolveVersion returns the Python version from flag or context.
// Flag takes precedence over context value.
func resolveVersion(ctx context.Context, flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return VersionFromContext(ctx)
}
