package pocket

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Must panics if err is not nil.
func Must(err error) {
	if err != nil {
		panic(fmt.Sprintf("pocket: %v", err))
	}
}

// DetectByFile finds directories containing any of the specified files (e.g., "go.mod").
// Returns paths relative to git root, sorted alphabetically.
// Excludes .pocket directory and hidden directories.
// Each directory is returned only once, even if multiple marker files are found.
func DetectByFile(filenames ...string) []string {
	root := GitRoot()
	seen := make(map[string]bool)

	// Build a set of target filenames for efficient lookup.
	targets := make(map[string]bool, len(filenames))
	for _, f := range filenames {
		targets[f] = true
	}

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr // Intentionally continue walking when directory is inaccessible.
		}

		// Skip hidden directories and .pocket.
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
		}

		// Check for target file.
		if !d.IsDir() && targets[d.Name()] {
			rel, err := filepath.Rel(root, filepath.Dir(path))
			Must(err) // Should never fail - both paths from same WalkDir.
			if rel == "" {
				rel = "."
			}
			// Normalize to forward slashes for cross-platform consistency.
			rel = filepath.ToSlash(rel)
			seen[rel] = true
		}
		return nil
	})

	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

// DetectByExtension finds directories containing files with any of the specified extensions.
// Returns paths relative to git root, sorted alphabetically.
// Excludes .pocket directory and hidden directories.
// Each directory is returned only once, even if multiple matching files are found.
func DetectByExtension(extensions ...string) []string {
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
		}

		// Check for target extension.
		if !d.IsDir() {
			for _, ext := range extensions {
				if strings.HasSuffix(d.Name(), ext) {
					rel, err := filepath.Rel(root, filepath.Dir(path))
					Must(err) // Should never fail - both paths from same WalkDir.
					if rel == "" {
						rel = "."
					}
					// Normalize to forward slashes for cross-platform consistency.
					rel = filepath.ToSlash(rel)
					seen[rel] = true
					break
				}
			}
		}
		return nil
	})

	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}
