// Package scaffold generates the initial .pocket directory structure.
package scaffold

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/config.go.tmpl
var configTemplate string

//go:embed templates/main.go.tmpl
var mainTemplate string

//go:embed templates/gitignore.tmpl
var gitignoreTemplate string

//go:embed templates/tools_gomod.tmpl
var toolsGomodTemplate string

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

	// Create tools/go.mod to prevent go mod tidy from scanning downloaded tools.
	if err := EnsureToolsGomod(pocketDir); err != nil {
		return err
	}

	return nil
}

// EnsureToolsGomod creates or updates .pocket/tools/go.mod.
// The Go version is read from .pocket/go.mod to stay in sync.
// This prevents go mod tidy from scanning downloaded Go toolchains and other tools.
func EnsureToolsGomod(pocketDir string) error {
	toolsDir := filepath.Join(pocketDir, "tools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		return fmt.Errorf("creating tools directory: %w", err)
	}

	goVersion, err := goVersionFromMod(pocketDir)
	if err != nil {
		return fmt.Errorf("reading Go version: %w", err)
	}

	tmpl, err := template.New("tools_gomod").Parse(toolsGomodTemplate)
	if err != nil {
		return fmt.Errorf("parsing tools/go.mod template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct{ GoVersion string }{GoVersion: goVersion}); err != nil {
		return fmt.Errorf("executing tools/go.mod template: %w", err)
	}

	gomodPath := filepath.Join(toolsDir, "go.mod")
	if err := os.WriteFile(gomodPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing tools/go.mod: %w", err)
	}
	return nil
}

// goVersionFromMod reads the Go version directive from go.mod in the specified directory.
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

// RegenerateMain regenerates only the main.go file.
// Useful when updating pocket without touching user's config.go.
func RegenerateMain(pocketDir string) error {
	mainPath := filepath.Join(pocketDir, "main.go")
	if err := os.WriteFile(mainPath, []byte(mainTemplate), 0o644); err != nil {
		return fmt.Errorf("writing main.go: %w", err)
	}

	// Ensure tools/go.mod exists (for older projects that don't have it).
	if err := EnsureToolsGomod(pocketDir); err != nil {
		return err
	}

	return nil
}
