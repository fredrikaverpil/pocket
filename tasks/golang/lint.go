package golang

import (
	"context"
	"flag"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/pcontext"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
)

var (
	lintFlags  = flag.NewFlagSet("go-lint", flag.ContinueOnError)
	lintFix    = lintFlags.Bool("fix", true, "apply fixes")
	lintConfig = lintFlags.String("config", "", "path to golangci-lint config file")
)

// Lint runs golangci-lint on Go code.
// Automatically installs golangci-lint if not present.
var Lint = pk.NewTask("go-lint", "run golangci-lint", lintFlags,
	pk.Serial(golangcilint.Install, lintCmd()),
)

func lintCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		args := []string{"run"}
		if pcontext.Verbose(ctx) {
			args = append(args, "-v")
		}
		if *lintConfig != "" {
			args = append(args, "-c", *lintConfig)
		}
		if *lintFix {
			args = append(args, "--fix")
		}
		args = append(args, "./...")
		return pk.Exec(ctx, golangcilint.Name, args...)
	})
}
