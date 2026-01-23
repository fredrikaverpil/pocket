package pk

import "context"

// runGitDiff runs `git diff --exit-code` after task execution.
// Git diff only runs when the -g flag is passed.
// Returns nil if git diff passes or is skipped, error if there are uncommitted changes.
func runGitDiff(ctx context.Context) error {
	if !gitDiffEnabledFromContext(ctx) {
		return nil
	}

	Printf(ctx, ":: git-diff\n")
	return Exec(ctx, "git", "diff", "--exit-code")
}
