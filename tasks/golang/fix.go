package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket"
)

// Fix runs go fix to update code for newer Go versions.
var Fix = pocket.Task("go-fix", "update code for newer Go versions", fixCmd())

func fixCmd() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		args := []string{"fix"}
		if pocket.Verbose(ctx) {
			args = append(args, "-v")
		}
		args = append(args, "./...")
		return pocket.Exec(ctx, "go", args...)
	})
}
