package python

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// SyncOptions configures the py-sync task.
type SyncOptions struct {
	PythonVersion string `arg:"python" usage:"Python version to use (e.g., 3.12)"`
}

// Sync installs Python dependencies using uv sync.
var Sync = pocket.Task("py-sync", "install Python dependencies",
	pocket.Serial(uv.Install, syncCmd()),
	pocket.Opts(SyncOptions{}),
)

func syncCmd() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		opts := pocket.Options[SyncOptions](ctx)

		// Use --all-groups to install dev dependencies (ruff, mypy, pytest, etc.)
		args := []string{"sync", "--all-groups"}
		if pocket.Verbose(ctx) {
			args = append(args, "--verbose")
		}
		if opts.PythonVersion != "" {
			args = append(args, "--python", opts.PythonVersion)
		}

		return pocket.Exec(ctx, uv.Name, args...)
	})
}
