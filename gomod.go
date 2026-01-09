package pocket

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GoVersionFromDir reads the Go version from go.mod in the given directory.
// Returns the version string (e.g., "1.21") or an error if the file
// cannot be read or doesn't contain a go directive.
func GoVersionFromDir(dir string) (string, error) {
	gomodPath := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}

	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "go "); ok {
			return strings.TrimSpace(after), nil
		}
	}
	return "", fmt.Errorf("no go directive in %s", gomodPath)
}
