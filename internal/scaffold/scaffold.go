// Package scaffold provides generation of .pocket/ scaffold files.
package scaffold

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	pocket "github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/internal/shim"
)

//go:embed main.go.tmpl
var MainTemplate []byte

//go:embed config.go.tmpl
var ConfigTemplate []byte

//go:embed gitignore.tmpl
var GitignoreTemplate []byte

// GenerateAll regenerates all generated files.
// Creates one-time files (config.go, .gitignore) if they don't exist.
// Always regenerates main.go and shim.
func GenerateAll(cfg *pocket.Config) error {
	pocketDir := filepath.Join(pocket.FromGitRoot(), pocket.DirName)

	// Ensure .pocket/ exists
	if err := os.MkdirAll(pocketDir, 0o755); err != nil {
		return fmt.Errorf("creating .pocket/: %w", err)
	}

	// Create config.go if not exists (user-editable, never overwritten)
	configPath := filepath.Join(pocketDir, "config.go")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, ConfigTemplate, 0o644); err != nil {
			return fmt.Errorf("writing config.go: %w", err)
		}
	}

	// Create .gitignore if not exists
	gitignorePath := filepath.Join(pocketDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		if err := os.WriteFile(gitignorePath, GitignoreTemplate, 0o644); err != nil {
			return fmt.Errorf("writing .gitignore: %w", err)
		}
	}

	// Always regenerate main.go
	if err := GenerateMain(); err != nil {
		return err
	}

	// Always regenerate shim(s).
	// Use provided config or a minimal default for initial scaffold.
	shimCfg := pocket.Config{}
	if cfg != nil {
		shimCfg = *cfg
	}
	if err := shim.Generate(shimCfg); err != nil {
		return err
	}

	return nil
}

// GenerateMain creates or updates .pocket/main.go from the template.
func GenerateMain() error {
	mainPath := filepath.Join(pocket.FromGitRoot(), pocket.DirName, "main.go")
	if err := os.WriteFile(mainPath, MainTemplate, 0o644); err != nil {
		return fmt.Errorf("writing .pocket/main.go: %w", err)
	}
	return nil
}
