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
		"race":    {Default: true, Usage: "enable race detector"},
		"run":     {Default: "", Usage: "run only tests matching regexp"},
		"timeout": {Default: "", Usage: "test timeout (e.g., 5m, 30s)"},
	},
	Do: func(ctx context.Context) error {
		args := []string{"test"}
		if pk.GetFlag[bool](ctx, "race") {
			args = append(args, "-race")
		}
		if pattern := pk.GetFlag[string](ctx, "run"); pattern != "" {
			args = append(args, "-run", pattern)
		}
		if timeout := pk.GetFlag[string](ctx, "timeout"); timeout != "" {
			args = append(args, "-timeout", timeout)
		}
		args = append(args, "./...")
		return pk.Exec(ctx, "go", args...)
	},
}
