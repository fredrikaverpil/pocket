package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
)

// Lint runs golangci-lint on Go code.
// Automatically installs golangci-lint if not present.
var Lint = &pk.Task{
	Name:  "go-lint",
	Usage: "run golangci-lint",
	Flags: map[string]pk.FlagDef{
		"config": {Default: "", Usage: "path to golangci-lint config file"},
		"fix":    {Default: true, Usage: "apply fixes"},
	},
	Body: pk.Serial(golangcilint.Install, lintCmd()),
}

func lintCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		args := []string{"run"}
		if pk.Verbose(ctx) {
			args = append(args, "-v")
		}
		if config := pk.GetFlag[string](ctx, "config"); config != "" {
			args = append(args, "-c", config)
		}
		if pk.GetFlag[bool](ctx, "fix") {
			args = append(args, "--fix")
		}
		args = append(args, "./...")
		return pk.Exec(ctx, golangcilint.Name, args...)
	})
}
