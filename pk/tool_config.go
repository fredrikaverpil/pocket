package pk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// ToolConfig describes how to find or create a tool's configuration file.
type ToolConfig struct {
	// UserFiles are filenames to search for in the repo root.
	// Checked in order; first match wins.
	UserFiles []string

	// DefaultFile is the filename for the bundled default config,
	// written to .pocket/tools/<name>/ if no user config exists.
	DefaultFile string

	// DefaultData is the bundled default configuration content.
	DefaultData []byte
}

// ConfigPath returns the path to a tool's config file.
//
// For each path in UserFiles:
//   - Absolute paths are checked as-is
//   - Relative paths are checked in the task's current directory (from PathFromContext)
//
// If no user config is found, writes DefaultData to .pocket/tools/<name>/<DefaultFile>.
// Returns empty string and no error if cfg is empty.
func ConfigPath(ctx context.Context, toolName string, cfg ToolConfig) (string, error) {
	if len(cfg.UserFiles) == 0 && cfg.DefaultFile == "" {
		return "", nil
	}

	// Check for user config files.
	cwd := FromGitRoot(PathFromContext(ctx))
	for _, configName := range cfg.UserFiles {
		var configPath string
		if filepath.IsAbs(configName) {
			configPath = configName
		} else {
			configPath = filepath.Join(cwd, configName)
		}
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	// No default config provided, return empty.
	if cfg.DefaultFile == "" || len(cfg.DefaultData) == 0 {
		return "", nil
	}

	// Write bundled config to .pocket/tools/<name>/<default-file>.
	configDir := FromToolsDir(toolName)
	configPath := filepath.Join(configDir, cfg.DefaultFile)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			return "", fmt.Errorf("create config dir: %w", err)
		}
		if err := os.WriteFile(configPath, cfg.DefaultData, 0o644); err != nil {
			return "", fmt.Errorf("write default config: %w", err)
		}
	}

	return configPath, nil
}
