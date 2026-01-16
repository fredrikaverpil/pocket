package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/govulncheck"
)

// Vulncheck runs govulncheck for vulnerability scanning.
var Vulncheck = pocket.Task("go-vulncheck", "run govulncheck", pocket.Serial(
	govulncheck.Install,
	vulncheckCmd(),
))

func vulncheckCmd() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		args := []string{}
		if pocket.Verbose(ctx) {
			args = append(args, "-show", "verbose")
		}
		args = append(args, "./...")
		return pocket.Exec(ctx, govulncheck.Name, args...)
	})
}
