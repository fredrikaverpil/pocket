// Package scaffold generates the initial .pocket directory structure.
package scaffold

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed templates/config.go.tmpl
var configTemplate string

//go:embed templates/main.go.tmpl
var mainTemplate string

//go:embed templates/gitignore.tmpl
var gitignoreTemplate string

// GenerateAll creates scaffold files in the .pocket directory.
// One-time files (config.go, .gitignore) are only created if missing.
// Auto-generated files (main.go) are always regenerated.
//
// Parameters:
//   - pocketDir: Absolute path to the .pocket directory.
func GenerateAll(pocketDir string) error {
	// Ensure .pocket directory exists.
	if err := os.MkdirAll(pocketDir, 0o755); err != nil {
		return fmt.Errorf("creating .pocket directory: %w", err)
	}

	// One-time files: only create if missing.
	oneTimeFiles := []struct {
		name    string
		content string
	}{
		{"config.go", configTemplate},
		{".gitignore", gitignoreTemplate},
	}

	for _, f := range oneTimeFiles {
		path := filepath.Join(pocketDir, f.name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(f.content), 0o644); err != nil {
				return fmt.Errorf("writing %s: %w", f.name, err)
			}
		}
	}

	// Auto-generated files: always regenerate.
	mainPath := filepath.Join(pocketDir, "main.go")
	if err := os.WriteFile(mainPath, []byte(mainTemplate), 0o644); err != nil {
		return fmt.Errorf("writing main.go: %w", err)
	}

	return nil
}

// RegenerateMain regenerates only the main.go file.
// Useful when updating pocket without touching user's config.go.
func RegenerateMain(pocketDir string) error {
	mainPath := filepath.Join(pocketDir, "main.go")
	if err := os.WriteFile(mainPath, []byte(mainTemplate), 0o644); err != nil {
		return fmt.Errorf("writing main.go: %w", err)
	}
	return nil
}
