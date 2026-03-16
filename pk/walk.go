package pk

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

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

var (
	regexMu    sync.RWMutex
	regexCache = make(map[string]*regexp.Regexp)
)

// matchPattern checks if a path matches a regex pattern.
func matchPattern(path, pattern string) (bool, error) {
	regexMu.RLock()
	re, ok := regexCache[pattern]
	regexMu.RUnlock()
	if !ok {
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			return false, fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
		regexMu.Lock()
		regexCache[pattern] = re
		regexMu.Unlock()
	}
	return re.MatchString(path), nil
}
