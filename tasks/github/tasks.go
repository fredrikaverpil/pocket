package github

import "github.com/fredrikaverpil/pocket/pk"

// Tasks returns GitHub-related tasks composed together.
// Currently returns only the Workflows task.
//
// Use directly or with pk.WithOptions for customization:
//
//	github.Tasks()
//
//	pk.WithOptions(
//	    github.Tasks(),
//	    pk.WithFlag(github.Workflows, "skip-pr", true),
//	)
func Tasks() pk.Runnable {
	return Workflows
}
