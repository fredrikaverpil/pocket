package github

import "github.com/fredrikaverpil/pocket/pk"

// Tasks returns GitHub-related tasks.
// Workflows generates GitHub Actions workflow files, including the per-job
// pocket-perjob workflow when enabled with -include-pocket-perjob flag.
//
// Use with pk.WithOptions to configure:
//
//	pk.WithOptions(
//	    github.Tasks(),
//	    pk.WithFlags(github.Workflows, github.WorkflowFlags{SkipPocket: true}),
//	    pk.WithContextValue(github.PerJobConfigKey{}, github.PerJobConfig{...}),
//	)
func Tasks() pk.Runnable {
	return Workflows
}
