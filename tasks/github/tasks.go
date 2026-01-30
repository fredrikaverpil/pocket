package github

import "github.com/fredrikaverpil/pocket/pk"

// Tasks returns GitHub-related tasks.
// Workflows generates GitHub Actions workflow files, including the static
// pocket-matrix workflow when enabled with -include-pocket-matrix flag.
//
// Use with pk.WithOptions to configure:
//
//	pk.WithOptions(
//	    github.Tasks(),
//	    pk.WithFlag(github.Workflows, "skip-pocket", true),
//	    pk.WithFlag(github.Workflows, "include-pocket-matrix", true),
//	    pk.WithContextValue(github.MatrixConfigKey{}, github.MatrixConfig{...}),
//	)
func Tasks() pk.Runnable {
	return Workflows
}
