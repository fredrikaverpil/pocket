package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/pcontext"
	"github.com/fredrikaverpil/pocket/tools/govulncheck"
)

// Vulncheck runs govulncheck for vulnerability scanning.
var Vulncheck = pk.NewTask("go-vulncheck", "run govulncheck", nil,
	pk.Serial(govulncheck.Install, vulncheckCmd()),
)

func vulncheckCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		args := []string{}
		if pcontext.Verbose(ctx) {
			args = append(args, "-show", "verbose")
		}
		args = append(args, "./...")
		return pk.Exec(ctx, govulncheck.Name, args...)
	})
}
