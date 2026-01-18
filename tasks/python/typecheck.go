package python

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// TypecheckOptions configures the py-typecheck task.
type TypecheckOptions struct {
	PythonVersion string `arg:"python" usage:"Python version to type-check against (e.g., 3.9)"`
}

// Typecheck type-checks Python files using mypy.
// Requires mypy as a project dependency in pyproject.toml.
var Typecheck = pocket.Task("py-typecheck", "type-check Python files",
	pocket.Serial(uv.Install, typecheckSyncCmd(), typecheckCmd()),
	pocket.Opts(TypecheckOptions{}),
)

func typecheckSyncCmd() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		opts := pocket.Options[TypecheckOptions](ctx)
		args := []string{"sync", "--all-groups"}
		if opts.PythonVersion != "" {
			args = append(args, "--python", opts.PythonVersion)
		}
		return pocket.Exec(ctx, uv.Name, args...)
	})
}

func typecheckCmd() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		opts := pocket.Options[TypecheckOptions](ctx)

		args := []string{}
		if pocket.Verbose(ctx) {
			args = append(args, "-v")
		}
		if opts.PythonVersion != "" {
			args = append(args, "--python-version", opts.PythonVersion)
		}
		args = append(args, pocket.Path(ctx))

		return uv.Run(ctx, opts.PythonVersion, "mypy", args...)
	})
}
