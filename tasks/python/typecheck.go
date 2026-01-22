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

func typecheckSyncCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return uv.Sync(ctx, *typecheckPyVer, true)
	})
}

func typecheckCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		args := []string{
			"--exclude", `\.pocket/`,
		}
		if pk.Verbose(ctx) {
			args = append(args, "-v")
		}
		if *typecheckPyVer != "" {
			args = append(args, "--python-version", *typecheckPyVer)
		}
		args = append(args, pk.PathFromContext(ctx))

		return uv.Run(ctx, *typecheckPyVer, "mypy", args...)
	})
}
