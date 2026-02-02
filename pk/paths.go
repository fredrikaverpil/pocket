package pk

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var (
	gitRootOnce  sync.Once
	gitRootCache string

	// findGitRootFunc allows overriding findGitRoot for tests.
	findGitRootFunc func() string
)

// findGitRoot walks up from the current directory to find the git repository root.
// Returns "." if not in a git repository.
// The result is cached after the first call for performance.
func findGitRoot() string {
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
	gitRoot := findGitRoot()
	parts := append([]string{gitRoot}, paths...)
	return filepath.Join(parts...)
}

// FromPocketDir constructs an absolute path within the .pocket directory.
//
//	FromPocketDir()           → "/path/to/repo/.pocket"
//	FromPocketDir("bin")      → "/path/to/repo/.pocket/bin"
//	FromPocketDir("tools")    → "/path/to/repo/.pocket/tools"
func FromPocketDir(elem ...string) string {
	parts := append([]string{findGitRoot(), ".pocket"}, elem...)
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

// walkDirectories walks the filesystem starting from gitRoot and returns
// all directories found (relative to gitRoot, using forward slashes).
// Skips directories in skipDirs, and hidden directories unless includeHidden is true.
func walkDirectories(gitRoot string, skipDirs []string, includeHidden bool) ([]string, error) {
	// Build a set for O(1) lookup
	skipSet := make(map[string]struct{}, len(skipDirs))
	for _, d := range skipDirs {
		skipSet[d] = struct{}{}
	}

	var dirs []string

	// Always include "." (the git root itself)
	dirs = append(dirs, ".")

	err := filepath.WalkDir(gitRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(gitRoot, path)
		if err != nil {
			return err
		}

		// Skip root (already added as ".")
		if relPath == "." {
			return nil
		}

		// Normalize to forward slashes
		relPath = filepath.ToSlash(relPath)

		base := filepath.Base(path)

		// Skip hidden directories unless includeHidden is true
		if !includeHidden && strings.HasPrefix(base, ".") {
			return filepath.SkipDir
		}

		// Skip directories in the skip set
		if _, skip := skipSet[base]; skip {
			return filepath.SkipDir
		}

		dirs = append(dirs, relPath)
		return nil
	})

	return dirs, err
}

// matchPattern checks if a path matches a regex pattern.
func matchPattern(path, pattern string) bool {
	matched, _ := regexp.MatchString(pattern, path)
	return matched
}
