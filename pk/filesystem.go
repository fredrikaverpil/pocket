package pk

import (
	"os"
	"path/filepath"
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
// Skips hidden directories, vendor, node_modules, and .pocket.
func walkDirectories(gitRoot string) ([]string, error) {
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

		// Skip hidden directories (except .git for the check, but we skip it anyway)
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") {
			return filepath.SkipDir
		}

		// Skip vendor and node_modules
		if base == "vendor" || base == "node_modules" {
			return filepath.SkipDir
		}

		dirs = append(dirs, relPath)
		return nil
	})

	return dirs, err
}

// matchPattern checks if a path matches a pattern.
// For now, implements simple prefix matching:
// - "services" matches "services", "services/api", "services/auth", etc.
// - "pkg" matches "pkg", "pkg/common", etc.
// Returns true if path matches the pattern.
func matchPattern(path, pattern string) bool {
	// Exact match
	if path == pattern {
		return true
	}

	// Prefix match (pattern is a directory prefix)
	if strings.HasPrefix(path, pattern+"/") {
		return true
	}

	return false
}
