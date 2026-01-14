// Package scaffold provides generation of .pocket/ scaffold files.
package scaffold

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	pocket "github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/internal/shim"
)

func init() {
	// Register GenerateAll with the pocket package to avoid import cycles.
	// This allows runner.go to call GenerateAll without importing this package.
	pocket.RegisterGenerateAll(GenerateAll)
}

//go:embed main.go.tmpl
var MainTemplate []byte

//go:embed config.go.tmpl
var ConfigTemplate []byte

//go:embed gitignore.tmpl
var GitignoreTemplate []byte

//go:embed tools_go.mod.tmpl
var toolsGoModTemplate string

// GenerateAll regenerates all generated files.
// Creates one-time files (config.go, .gitignore) if they don't exist.
// Always regenerates main.go and shim.
// Returns the list of generated shim paths relative to the git root.
func GenerateAll(cfg *pocket.Config) ([]string, error) {
	pocketDir := filepath.Join(pocket.FromGitRoot(), pocket.DirName)

	// Ensure .pocket/ exists
	if err := os.MkdirAll(pocketDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating .pocket/: %w", err)
	}

	// Create config.go if not exists (user-editable, never overwritten)
	configPath := filepath.Join(pocketDir, "config.go")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, ConfigTemplate, 0o644); err != nil {
			return nil, fmt.Errorf("writing config.go: %w", err)
		}
	}

	// Create .gitignore if not exists
	gitignorePath := filepath.Join(pocketDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		if err := os.WriteFile(gitignorePath, GitignoreTemplate, 0o644); err != nil {
			return nil, fmt.Errorf("writing .gitignore: %w", err)
		}
	}

	// Always regenerate main.go
	if err := GenerateMain(); err != nil {
		return nil, err
	}

	// Generate tools/go.mod if tools directory exists (prevents go mod tidy issues)
	toolsDir := pocket.FromToolsDir()
	if _, err := os.Stat(toolsDir); err == nil {
		if err := GenerateToolsGoMod(); err != nil {
			return nil, err
		}
	}

	// Always regenerate shim(s).
	// Use provided config or a minimal default for initial scaffold.
	shimCfg := pocket.Config{}
	if cfg != nil {
		shimCfg = *cfg
	}
	shimPaths, err := shim.Generate(shimCfg)
	if err != nil {
		return nil, err
	}

	return shimPaths, nil
}

// GenerateMain creates or updates .pocket/main.go from the template.
func GenerateMain() error {
	mainPath := filepath.Join(pocket.FromGitRoot(), pocket.DirName, "main.go")
	if err := os.WriteFile(mainPath, MainTemplate, 0o644); err != nil {
		return fmt.Errorf("writing .pocket/main.go: %w", err)
	}
	return nil
}

// GenerateToolsGoMod creates .pocket/tools/go.mod if it doesn't exist.
// This prevents `go mod tidy` in .pocket/ from scanning downloaded tools
// (like Go SDK test files) which contain relative imports that break module mode.
func GenerateToolsGoMod() error {
	toolsDir := pocket.FromToolsDir()
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		return fmt.Errorf("creating tools dir: %w", err)
	}

	goModPath := filepath.Join(toolsDir, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		return nil // Already exists
	}

	// Read Go version from .pocket/go.mod
	goVersion, err := pocket.GoVersionFromDir(pocket.FromPocketDir())
	if err != nil {
		return err
	}

	tmpl, err := template.New("tools_go.mod").Parse(toolsGoModTemplate)
	if err != nil {
		return fmt.Errorf("parsing tools go.mod template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{"GoVersion": goVersion}); err != nil {
		return fmt.Errorf("executing tools go.mod template: %w", err)
	}

	if err := os.WriteFile(goModPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing tools/go.mod: %w", err)
	}
	return nil
}
