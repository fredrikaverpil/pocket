package python

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/run"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// FormatFlags holds flags for the Format task.
type FormatFlags struct {
	Python string `flag:"python" usage:"Python version (for target-version inference)"`
}

// Format formats Python files using ruff format.
// Requires ruff as a project dependency in pyproject.toml.
// Python version can be set via the -python flag.
var Format = &pk.Task{
	Name:  "py-format",
	Usage: "format Python files",
	Flags: FormatFlags{},
	Body:  pk.Serial(uv.Install, formatSyncCmd(), formatCmd()),
}

func formatSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := run.GetFlags[FormatFlags](ctx)
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: f.Python,
			AllGroups:     true,
		})
	})
}

func formatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := run.GetFlags[FormatFlags](ctx)
		return runFormat(ctx, f.Python)
	})
}

func runFormat(ctx context.Context, pythonVersion string) error {
	args := []string{
		"format",
		"--exclude", ".pocket",
	}
	if run.Verbose(ctx) {
		args = append(args, "--verbose")
	}
	if pythonVersion != "" {
		args = append(args, "--target-version", pythonVersionToRuff(pythonVersion))
	}
	args = append(args, run.PathFromContext(ctx))

	return uv.Run(ctx, uv.RunOptions{PythonVersion: pythonVersion}, "ruff", args...)
}
