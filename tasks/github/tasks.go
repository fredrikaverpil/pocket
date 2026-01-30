package github

import "github.com/fredrikaverpil/pocket/pk"

// Tasks returns GitHub-related tasks: Workflows and Matrix.
// Workflows generates GitHub Actions workflow files.
// Matrix generates the matrix JSON for pocket-matrix.yml.
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
	return pk.Serial(Workflows, Matrix)
}
