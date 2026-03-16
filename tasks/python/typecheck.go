package python

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/run"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// TypecheckFlags holds flags for the Typecheck task.
type TypecheckFlags struct {
	Python string `flag:"python" usage:"Python version to type-check against (e.g., 3.9)"`
}

// Typecheck type-checks Python files using mypy.
// Requires mypy as a project dependency in pyproject.toml.
// Python version can be set via the -python flag.
var Typecheck = &pk.Task{
	Name:  "py-typecheck",
	Usage: "type-check Python files",
	Flags: TypecheckFlags{},
	Body:  pk.Serial(uv.Install, typecheckSyncCmd(), typecheckCmd()),
}

func typecheckSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := run.GetFlags[TypecheckFlags](ctx)
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: f.Python,
			AllGroups:     true,
		})
	})
}

func typecheckCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := run.GetFlags[TypecheckFlags](ctx)
		return runTypecheck(ctx, f.Python)
	})
}

func runTypecheck(ctx context.Context, pythonVersion string) error {
	args := []string{
		"--exclude", `\.pocket/`,
	}
	if run.Verbose(ctx) {
		args = append(args, "-v")
	}
	if pythonVersion != "" {
		args = append(args, "--python-version", pythonVersion)
	}
	args = append(args, run.PathFromContext(ctx))

	return uv.Run(ctx, uv.RunOptions{PythonVersion: pythonVersion}, "mypy", args...)
}
