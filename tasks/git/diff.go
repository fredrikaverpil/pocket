package git

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
)

// Diff is a task that checks for uncommitted changes in the repository.
// It fails if the git workspace is dirty.
var Diff = pk.NewTask("git-diff", "check for uncommitted changes", nil, pk.Do(func(ctx context.Context) error {
	return pk.Exec(ctx, "git", "diff", "--exit-code")
}))
