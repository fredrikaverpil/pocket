package pk

import (
	"context"
	"errors"
)

// ErrGitDiffUncommitted is returned when git diff detects uncommitted changes.
var ErrGitDiffUncommitted = errors.New("uncommitted changes detected")

// runGitDiff runs `git diff --exit-code` after task execution.
// Git diff only runs when the -g flag is passed.
// Returns nil if git diff passes or is skipped, ErrGitDiffUncommitted if there are changes.
func runGitDiff(ctx context.Context) error {
	if !gitDiffEnabledFromContext(ctx) {
		return nil
	}

	Printf(ctx, ":: git-diff\n")
	if err := Exec(ctx, "git", "diff", "--exit-code"); err != nil {
		return ErrGitDiffUncommitted
	}
	return nil
}
