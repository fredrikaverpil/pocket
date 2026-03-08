package github

import "github.com/fredrikaverpil/pocket/pk"

// Tasks returns GitHub-related tasks.
// Workflows generates GitHub Actions workflow files, including the per-job
// pocket-perjob workflow when enabled with the PerPocketTaskJob flag.
//
// Use with pk.WithOptions to configure:
//
//	pk.WithOptions(
//	    github.Tasks(),
//	    pk.WithFlags(github.WorkflowFlags{PerPocketTaskJob: true}),
//	)
func Tasks() pk.Runnable {
	return Workflows
}
