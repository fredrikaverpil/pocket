package python

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// FormatOptions configures the py-format task.
type FormatOptions struct {
	PythonVersion string `arg:"python"      usage:"Python version (for target-version inference)"`
}

// Format formats Python files using ruff format.
// Requires ruff as a project dependency in pyproject.toml.
var Format = pocket.Task("py-format", "format Python files",
	pocket.Serial(uv.Install, formatSyncCmd(), formatCmd()),
	pocket.Opts(FormatOptions{}),
)

func formatSyncCmd() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		opts := pocket.Options[FormatOptions](ctx)
		args := []string{"sync", "--all-groups"}
		if opts.PythonVersion != "" {
			args = append(args, "--python", opts.PythonVersion)
		}
		return pocket.Exec(ctx, uv.Name, args...)
	})
}

func formatCmd() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		opts := pocket.Options[FormatOptions](ctx)

		args := []string{"format"}
		if pocket.Verbose(ctx) {
			args = append(args, "--verbose")
		}
		if opts.PythonVersion != "" {
			args = append(args, "--target-version", pythonVersionToRuff(opts.PythonVersion))
		}
		args = append(args, pocket.Path(ctx))

		return uv.Run(ctx, opts.PythonVersion, "ruff", args...)
	})
}
