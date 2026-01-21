package pk

import (
	"os"
	"path/filepath"
)

// DetectFunc is a function that filters directories to find those with specific markers.
// It receives the pre-walked directory list and git root path, and returns matching directories.
// Used with WithDetect to dynamically discover paths for task execution.
type DetectFunc func(dirs []string, gitRoot string) []string

// DetectByFile returns a DetectFunc that finds directories containing any of the specified files.
// For example, DetectByFile("go.mod") finds all Go modules.
func DetectByFile(filenames ...string) DetectFunc {
	return func(dirs []string, gitRoot string) []string {
		var result []string
		for _, dir := range dirs {
			absDir := filepath.Join(gitRoot, dir)
			for _, filename := range filenames {
				path := filepath.Join(absDir, filename)
				if _, err := os.Stat(path); err == nil {
					result = append(result, dir)
					break // Found a match, no need to check other filenames
				}
			}
		}
		return result
	}
}
