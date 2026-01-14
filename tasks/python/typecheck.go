package python

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/mypy"
)

// Typecheck type-checks Python files using mypy.
var Typecheck = pocket.Func("py-typecheck", "type-check Python files", typecheck)

func typecheck(ctx context.Context) error {
	pocket.Serial(ctx, mypy.Install)
	return pocket.Exec(ctx, mypy.Name, pocket.Path(ctx))
}
