package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
)

// Flag names for the Lint task.
const (
	FlagLintConfig = "config"
	FlagLintFix    = "fix"
)

// Lint runs golangci-lint on Go code.
// Automatically installs golangci-lint if not present.
var Lint = &pk.Task{
	Name:  "go-lint",
	Usage: "run golangci-lint",
	Flags: map[string]pk.FlagDef{
		FlagLintConfig: {Default: "", Usage: "path to golangci-lint config file"},
		FlagLintFix:    {Default: true, Usage: "apply fixes"},
	},
	Body: pk.Serial(golangcilint.Install, lintCmd()),
}

func lintCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		args := []string{"run"}
		if pk.Verbose(ctx) {
			args = append(args, "-v")
		}
		if config := pk.GetFlag[string](ctx, FlagLintConfig); config != "" {
			args = append(args, "-c", config)
		}
		if pk.GetFlag[bool](ctx, FlagLintFix) {
			args = append(args, "--fix")
		}
		args = append(args, "./...")
		return pk.Exec(ctx, golangcilint.Name, args...)
	})
}
