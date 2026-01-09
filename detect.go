package pocket

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// detectDirs walks the git repository and returns directories containing files
// that match the predicate. Excludes hidden directories and common vendor directories.
// Returns paths relative to git root, sorted alphabetically.
func detectDirs(predicate func(name string) bool) []string {
	root := GitRoot()
	seen := make(map[string]bool)

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr // Intentionally continue walking when directory is inaccessible.
		}

		// Skip hidden directories and common vendor directories.
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches predicate.
		if predicate(d.Name()) {
			rel, _ := filepath.Rel(root, filepath.Dir(path))
			if rel == "" {
				rel = "."
			}
			// Normalize to forward slashes for cross-platform consistency.
			seen[filepath.ToSlash(rel)] = true
		}
		return nil
	})

	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	return paths
}

// DetectByFile finds directories containing any of the specified files (e.g., "go.mod").
// Returns paths relative to git root, sorted alphabetically.
// Excludes .pocket directory and hidden directories.
// Each directory is returned only once, even if multiple marker files are found.
func DetectByFile(filenames ...string) []string {
	targets := make(map[string]bool, len(filenames))
	for _, f := range filenames {
		targets[f] = true
	}
	return detectDirs(func(name string) bool {
		return targets[name]
	})
}

// DetectByExtension finds directories containing files with any of the specified extensions.
// Returns paths relative to git root, sorted alphabetically.
// Excludes .pocket directory and hidden directories.
// Each directory is returned only once, even if multiple matching files are found.
func DetectByExtension(extensions ...string) []string {
	return detectDirs(func(name string) bool {
		for _, ext := range extensions {
			if strings.HasSuffix(name, ext) {
				return true
			}
		}
		return false
	})
}
