package python

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/mypy"
)

// Typecheck type-checks Python files using mypy.
var Typecheck = pocket.Task("py-typecheck", "type-check Python files", pocket.Serial(
	mypy.Install,
	typecheckCmd(),
))

func typecheckCmd() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		args := []string{}
		if pocket.Verbose(ctx) {
			args = append(args, "-v")
		}
		args = append(args, pocket.Path(ctx))
		return pocket.Exec(ctx, mypy.Name, args...)
	})
}
