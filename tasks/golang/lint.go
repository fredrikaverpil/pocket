package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
)

// Lint runs golangci-lint on Go code.
// Automatically installs golangci-lint if not present.
var Lint = pk.DefineTask("go-lint", "run golangci-lint",
	pk.Serial(golangcilint.Install, lintCmd()),
)

func lintCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return pk.Exec(ctx, golangcilint.Name, "run", "--fix", "./...")
	})
}
