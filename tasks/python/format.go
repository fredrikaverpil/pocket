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
var Format = pk.NewTask("py-format", "format Python files", formatFlags,
	pk.Serial(uv.Install, formatSyncCmd(), formatCmd()),
)

// formatWith creates a format task for a specific Python version.
func formatWith(pythonVersion string) *pk.Task {
	return pk.NewTask("py-format:"+pythonVersion, "format Python files", nil,
		pk.Serial(uv.Install, formatSyncCmdWith(pythonVersion), formatCmdWith(pythonVersion)),
	)
}

func formatSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: *formatPyVer,
			AllGroups:     true,
		})
	})
}

func formatSyncCmdWith(pythonVersion string) pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: pythonVersion,
			AllGroups:     true,
		})
	})
}

func formatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return runFormat(ctx, *formatPyVer)
	})
}

func formatCmdWith(pythonVersion string) pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return runFormat(ctx, pythonVersion)
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
