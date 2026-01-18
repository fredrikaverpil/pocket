package pk

import "path/filepath"

// FromGitRoot constructs an absolute path by joining the given path segments
// relative to the git repository root.
// Returns an absolute path to the target location.
//
// Handles both forms:
//   FromGitRoot("services/api")        → "/path/to/repo/services/api"
//   FromGitRoot("services", "api")     → "/path/to/repo/services/api"
//   FromGitRoot("pkg")                 → "/path/to/repo/pkg"
//   FromGitRoot(".")                   → "/path/to/repo"
func FromGitRoot(paths ...string) string {
	gitRoot := findGitRoot()
	parts := append([]string{gitRoot}, paths...)
	return filepath.Join(parts...)
}
