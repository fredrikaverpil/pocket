package repopath

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	gitRootOnce  sync.Once
	gitRootCache string

	// findGitRootFunc allows overriding GitRoot for tests.
	findGitRootFunc func() string
)

// SetGitRootFunc overrides GitRoot for testing. Pass nil to restore default.
// This must only be used in tests.
func SetGitRootFunc(fn func() string) {
	findGitRootFunc = fn
}

// GitRoot walks up from the current directory to find the git repository root.
// Returns "." if not in a git repository.
// The result is cached after the first call for performance.
func GitRoot() string {
	if findGitRootFunc != nil {
		return findGitRootFunc()
	}
	gitRootOnce.Do(func() {
		gitRootCache = doFindGitRoot()
	})
	return gitRootCache
}

// doFindGitRoot performs the actual git root discovery.
func doFindGitRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	for {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding .git
			return "."
		}
		dir = parent
	}
}

// FromGitRoot constructs an absolute path by joining the given path segments
// relative to the git repository root.
// Returns an absolute path to the target location.
//
// Handles both forms:
//
//	FromGitRoot("services/api")        → "/path/to/repo/services/api"
//	FromGitRoot("services", "api")     → "/path/to/repo/services/api"
//	FromGitRoot("pkg")                 → "/path/to/repo/pkg"
//	FromGitRoot(".")                   → "/path/to/repo"
func FromGitRoot(paths ...string) string {
	gitRoot := GitRoot()
	parts := append([]string{gitRoot}, paths...)
	return filepath.Join(parts...)
}

// FromPocketDir constructs an absolute path within the .pocket directory.
//
//	FromPocketDir()           → "/path/to/repo/.pocket"
//	FromPocketDir("bin")      → "/path/to/repo/.pocket/bin"
//	FromPocketDir("tools")    → "/path/to/repo/.pocket/tools"
func FromPocketDir(elem ...string) string {
	parts := append([]string{GitRoot(), ".pocket"}, elem...)
	return filepath.Join(parts...)
}

// FromToolsDir constructs an absolute path within .pocket/tools.
//
//	FromToolsDir()                    → "/path/to/repo/.pocket/tools"
//	FromToolsDir("go", "pkg", "v1.0") → "/path/to/repo/.pocket/tools/go/pkg/v1.0"
func FromToolsDir(elem ...string) string {
	parts := append([]string{"tools"}, elem...)
	return FromPocketDir(parts...)
}

// FromBinDir constructs an absolute path within .pocket/bin.
//
//	FromBinDir()             → "/path/to/repo/.pocket/bin"
//	FromBinDir("golangci-lint") → "/path/to/repo/.pocket/bin/golangci-lint"
func FromBinDir(elem ...string) string {
	parts := append([]string{"bin"}, elem...)
	return FromPocketDir(parts...)
}
