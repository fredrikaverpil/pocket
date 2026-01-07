package pocket

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

// ExtractGoVersion reads a go.mod file from the given module path
// and returns the Go version specified in the "go" directive.
// The modulePath should be relative to the git root (e.g., "." or "submodule").
func ExtractGoVersion(modulePath string) (string, error) {
	gomodPath := filepath.Join(FromGitRoot(modulePath), "go.mod")
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}

	f, err := modfile.ParseLax(gomodPath, data, nil)
	if err != nil {
		return "", fmt.Errorf("parse go.mod: %w", err)
	}

	if f.Go == nil {
		return "", fmt.Errorf("no go directive in %s", gomodPath)
	}

	return f.Go.Version, nil
}
