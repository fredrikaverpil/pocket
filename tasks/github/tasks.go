package github

import "github.com/fredrikaverpil/pocket/pk"

// matrixConfigKey is the context key for MatrixConfig.
type matrixConfigKey struct{}

// Tasks returns GitHub-related tasks composed together.
// Returns Workflows (auto) and Matrix (manual) tasks.
//
// Use with pk.WithOptions for customization:
//
//	pk.WithOptions(
//	    github.Tasks(),
//	    github.WithSkipPocket(),
//	    github.WithMatrixWorkflow(github.MatrixConfig{...}),
//	)
func Tasks() pk.Runnable {
	return pk.Parallel(
		Workflows,
		matrix().Manual(),
	)
}

// WithSkipPocket excludes the pocket.yml workflow.
func WithSkipPocket() pk.PathOption {
	return pk.WithFlag(Workflows, "skip-pocket", true)
}

// WithSkipPR excludes the pr.yml workflow.
func WithSkipPR() pk.PathOption {
	return pk.WithFlag(Workflows, "skip-pr", true)
}

// WithSkipRelease excludes the release.yml workflow.
func WithSkipRelease() pk.PathOption {
	return pk.WithFlag(Workflows, "skip-release", true)
}

// WithSkipStale excludes the stale.yml workflow.
func WithSkipStale() pk.PathOption {
	return pk.WithFlag(Workflows, "skip-stale", true)
}

// WithPlatforms sets the platforms for the pocket.yml workflow.
// Platforms should be comma-separated (e.g., "ubuntu-latest, macos-latest").
func WithPlatforms(platforms string) pk.PathOption {
	return pk.WithFlag(Workflows, "platforms", platforms)
}

// WithMatrixWorkflow enables the pocket-matrix.yml workflow and configures
// the gha-matrix task. The matrix task outputs JSON for GitHub Actions' fromJson().
//
// Example:
//
//	pk.WithOptions(
//	    github.Tasks(),
//	    github.WithMatrixWorkflow(github.MatrixConfig{
//	        DefaultPlatforms: []string{"ubuntu-latest", "macos-latest"},
//	        TaskOverrides: map[string]github.TaskOverride{
//	            "go-lint": {Platforms: []string{"ubuntu-latest"}},
//	        },
//	    }),
//	)
func WithMatrixWorkflow(cfg MatrixConfig) pk.PathOption {
	return pk.CombineOptions(
		pk.WithFlag(Workflows, "include-pocket-matrix", true),
		pk.WithContextValue(matrixConfigKey{}, cfg),
	)
}
