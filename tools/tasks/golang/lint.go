package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
)

// LintFlags holds flags for the Lint task.
type LintFlags struct {
	Config string `flag:"config" usage:"path to golangci-lint config file"`
	Fix    bool   `flag:"fix"    usage:"apply fixes"`
}

// Lint runs golangci-lint on Go code.
// Automatically installs golangci-lint if not present.
var Lint = &pk.Task{
	Name:  "go-lint",
	Usage: "run golangci-lint",
	Flags: LintFlags{Fix: true},
	Body:  pk.Serial(golangcilint.Install, lintCmd()),
}

func lintCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := pk.GetFlags[LintFlags](ctx)
		args := []string{"run"}
		if pk.Verbose(ctx) {
			args = append(args, "-v")
		}

		configPath := f.Config
		if configPath == "" && !golangcilint.HasProjectConfig() {
			configPath = golangcilint.EnsureDefaultConfig()
		}
		if configPath != "" {
			args = append(args, "-c", configPath)
		}

		if f.Fix {
			args = append(args, "--fix")
		}
		args = append(args, "./...")
		return pk.Exec(ctx, golangcilint.Name, args...)
	})
}
