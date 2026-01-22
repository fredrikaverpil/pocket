package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// goVersionFromMod reads the Go version directive ("go X.Y") from go.mod
// in the specified directory. Returns the version string (e.g., "1.25.5")
// or an error if the file cannot be read or doesn't contain a go directive.
func goVersionFromMod(dir string) (string, error) {
	gomodPath := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "go "); ok {
			return strings.TrimSpace(after), nil
		}
	}
	return "", fmt.Errorf("no go directive in %s", gomodPath)
}
