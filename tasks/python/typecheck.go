package python

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// Typecheck type-checks Python files using mypy.
// Requires mypy as a project dependency in pyproject.toml.
// Python version can be set via the -python flag.
var Typecheck = &pk.Task{
	Name:  "py-typecheck",
	Usage: "type-check Python files",
	Flags: map[string]pk.FlagDef{
		FlagPython: {Default: "", Usage: "Python version to type-check against (e.g., 3.9)"},
	},
	Body: pk.Serial(uv.Install, typecheckSyncCmd(), typecheckCmd()),
}

func typecheckSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := resolveVersion(ctx, pk.GetFlag[string](ctx, FlagPython))
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: version,
			AllGroups:     true,
		})
	})
}

func typecheckCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := resolveVersion(ctx, pk.GetFlag[string](ctx, FlagPython))
		return runTypecheck(ctx, version)
	})
}

func runTypecheck(ctx context.Context, pythonVersion string) error {
	args := []string{
		"--exclude", `\.pocket/`,
	}
	if pk.Verbose(ctx) {
		args = append(args, "-v")
	}
	if pythonVersion != "" {
		args = append(args, "--python-version", pythonVersion)
	}
	args = append(args, pk.PathFromContext(ctx))

	return uv.Run(ctx, uv.RunOptions{PythonVersion: pythonVersion}, "mypy", args...)
}
