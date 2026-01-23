package python

import (
	"context"
	"flag"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

var (
	typecheckFlags = flag.NewFlagSet("py-typecheck", flag.ContinueOnError)
	typecheckPyVer = typecheckFlags.String("python", "", "Python version to type-check against (e.g., 3.9)")
)

// Typecheck type-checks Python files using mypy.
// Requires mypy as a project dependency in pyproject.toml.
var Typecheck = pk.NewTask("py-typecheck", "type-check Python files", typecheckFlags,
	pk.Serial(uv.Install, typecheckSyncCmd(), typecheckCmd()),
)

// typecheckWith creates a typecheck task for a specific Python version.
func typecheckWith(pythonVersion string) *pk.Task {
	return pk.NewTask("py-typecheck:"+pythonVersion, "type-check Python files", nil,
		pk.Serial(uv.Install, typecheckSyncCmdWith(pythonVersion), typecheckCmdWith(pythonVersion)),
	)
}

func typecheckSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: *typecheckPyVer,
			AllGroups:     true,
		})
	})
}

func typecheckSyncCmdWith(pythonVersion string) pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: pythonVersion,
			AllGroups:     true,
		})
	})
}

func typecheckCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return runTypecheck(ctx, *typecheckPyVer)
	})
}

func typecheckCmdWith(pythonVersion string) pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return runTypecheck(ctx, pythonVersion)
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
