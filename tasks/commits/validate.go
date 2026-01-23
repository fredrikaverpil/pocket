package commits

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/commitsar"
)

// Validate validates commit messages against conventional commit standards.
var Validate = pk.NewTask("commits-validate", "validate commits against conventional commits", nil,
	pk.Serial(commitsar.Install, validateCmd()),
)

func validateCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return pk.Exec(ctx, commitsar.Name)
	})
}
