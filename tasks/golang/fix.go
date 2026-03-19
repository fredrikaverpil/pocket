package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/run"
)

// Fix runs go fix to update code for newer Go versions.
var Fix = &pk.Task{
	Name:  "go-fix",
	Usage: "update code for newer Go versions",
	Body: pk.Do(func(ctx context.Context) error {
		args := []string{"fix"}
		if run.Verbose(ctx) {
			args = append(args, "-v")
		}
		args = append(args, "./...")
		return run.Exec(ctx, "go", args...)
	}),
}
