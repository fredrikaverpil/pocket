package renovate

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/run"
	"github.com/fredrikaverpil/pocket/tools/bun"
)

// Validate validates the Renovate configuration.
var Validate = &pk.Task{
	Name:  "renovate-validate",
	Usage: "validate Renovate config",
	Body:  pk.Serial(bun.Install, validateCmd()),
}

func validateCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return run.Exec(ctx,
			bun.Name,
			"x",
			"--package",
			"renovate",
			"renovate-config-validator",
			"renovate.json",
		)
	})
}
