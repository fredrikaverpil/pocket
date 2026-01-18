package pk

import (
	"os"
	"path/filepath"
	"strings"
)

// findGitRoot walks up from the current directory to find the git repository root.
// Returns "." if not in a git repository.
func findGitRoot() string {
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

		// Skip vendor, node_modules, .pocket
		if base == "vendor" || base == "node_modules" || base == ".pocket" {
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

// resolvePathPatterns takes include/exclude patterns and returns
// the list of actual directories that match.
func resolvePathPatterns(gitRoot string, includePaths, excludePaths []string) ([]string, error) {
	// Walk filesystem to get all directories
	allDirs, err := walkDirectories(gitRoot)
	if err != nil {
		return nil, err
	}

	// If no include patterns, start with all directories
	var candidates []string
	if len(includePaths) == 0 {
		// No include patterns means "run everywhere" (just root)
		candidates = []string{"."}
	} else {
		// Filter by include patterns
		for _, dir := range allDirs {
			for _, pattern := range includePaths {
				if matchPattern(dir, pattern) {
					candidates = append(candidates, dir)
					break
				}
			}
		}
	}

	// Apply exclude patterns
	var result []string
	for _, dir := range candidates {
		excluded := false
		for _, pattern := range excludePaths {
			if matchPattern(dir, pattern) {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, dir)
		}
	}

	return result, nil
}
