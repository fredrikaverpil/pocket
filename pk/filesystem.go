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
)

// findGitRoot walks up from the current directory to find the git repository root.
// Returns "." if not in a git repository.
// The result is cached after the first call for performance.
func findGitRoot() string {
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
// Returns true if path matches the pattern.
func matchPattern(path, pattern string) bool {
	matched, _ := regexp.MatchString(pattern, path)
	return matched
}
