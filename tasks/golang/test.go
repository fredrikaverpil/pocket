package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
)

// Test runs Go tests.
var Test = &pk.Task{
	Name:  "go-test",
	Usage: "run go tests",
	Flags: map[string]pk.FlagDef{
		"race": {Default: true, Usage: "enable race detector"},
		"run":  {Default: "", Usage: "run only tests matching regexp"},
	},
	Do: func(ctx context.Context) error {
		args := []string{"test"}
		if pk.GetFlag[bool](ctx, "race") {
			args = append(args, "-race")
		}
		if pattern := pk.GetFlag[string](ctx, "run"); pattern != "" {
			args = append(args, "-run", pattern)
		}
		args = append(args, "./...")
		return pk.Exec(ctx, "go", args...)
	},
}
