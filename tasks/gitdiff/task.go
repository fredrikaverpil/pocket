// Package gitdiff provides a task that fails if there are uncommitted changes.
package gitdiff

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
)

// Task returns a task that runs git diff --exit-code.
// It fails if there are any uncommitted changes, ensuring all generated
// or formatted files have been committed.
func Task() *pocket.Task {
	return &pocket.Task{
		Name:    "git-diff",
		Usage:   "fail if there are uncommitted changes",
		Builtin: true,
		Action: func(ctx context.Context, _ map[string]string) error {
			cmd := pocket.Command(ctx, "git", "diff", "--exit-code")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("uncommitted changes detected; please commit or stage your changes")
			}
			return nil
		},
	}
}
