package repopath

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	// ErrGitRootNotFound is returned when no git repository root is found.
	ErrGitRootNotFound = errors.New("git root not found")

	gitRootOnce  sync.Once
	gitRootCache string
	gitRootErr   error

	// findGitRootFunc allows overriding GitRoot for tests.
	findGitRootFunc func() string
)

// SetGitRootFunc overrides GitRoot for testing and clears the cached root.
// Pass nil to restore default discovery. This must only be used in tests.
func SetGitRootFunc(fn func() string) {
	findGitRootFunc = fn
	gitRootOnce = sync.Once{}
	gitRootCache = ""
	gitRootErr = nil
}

// FindGitRoot walks up from the current directory to find the git repository root.
// It returns ErrGitRootNotFound when no repository root exists.
// The result is cached after the first call for performance.
func FindGitRoot() (string, error) {
	if findGitRootFunc != nil {
		return findGitRootFunc(), nil
	}
	gitRootOnce.Do(func() {
		gitRootCache, gitRootErr = doFindGitRoot()
	})
	return gitRootCache, gitRootErr
}

// GitRoot walks up from the current directory to find the git repository root.
// Returns "." if not in a git repository.
// The result is cached after the first call for performance.
func GitRoot() string {
	gitRoot, err := FindGitRoot()
	if err != nil {
		return "."
	}
	return gitRoot
}

// doFindGitRoot performs the actual git root discovery.
func doFindGitRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrGitRootNotFound
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
