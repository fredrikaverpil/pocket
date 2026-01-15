package python

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/mypy"
)

// Typecheck type-checks Python files using mypy.
var Typecheck = pocket.Func("py-typecheck", "type-check Python files", pocket.Serial(
	mypy.Install,
	typecheck,
))

func typecheck(ctx context.Context) error {
	return pocket.Exec(ctx, mypy.Name, pocket.Path(ctx))
}
