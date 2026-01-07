// Package gitdiff provides a task that fails if there are uncommitted changes.
package gitdiff

import (
	"github.com/fredrikaverpil/pocket"
	"github.com/goyek/goyek/v3"
)

// Task returns a goyek task that runs git diff --exit-code.
// It fails if there are any uncommitted changes, ensuring all generated
// or formatted files have been committed.
func Task() *goyek.DefinedTask {
	return goyek.Define(goyek.Task{
		Name:  "git-diff",
		Usage: "fail if there are uncommitted changes",
		Action: func(a *goyek.A) {
			cmd := pocket.Command(a.Context(), "git", "diff", "--exit-code")
			if err := cmd.Run(); err != nil {
				a.Fatal("uncommitted changes detected; please commit or stage your changes")
			}
		},
	})
}
