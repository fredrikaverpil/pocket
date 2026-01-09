// Package ruff provides ruff (Python linter and formatter) tool integration.
// ruff is installed via uv into a virtual environment.
package ruff

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tool"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

const name = "ruff"

// renovate: datasource=pypi depName=ruff
const version = "0.14.0"

// pythonVersion specifies the Python version for the virtual environment.
const pythonVersion = "3.12"

//go:embed ruff.toml
var defaultConfig []byte

var t = &tool.Tool{Name: name, Prepare: Prepare}

// Command prepares the tool and returns an exec.Cmd for running ruff.
var Command = t.Command

// Run installs (if needed) and executes ruff.
var Run = t.Run

// ConfigPath returns the path to the ruff config file.
// It checks for ruff.toml or pyproject.toml in the repo root first,
// then falls back to the bundled default config.
func ConfigPath() (string, error) {
	// Check for user config in repo root.
	for _, configName := range []string{"ruff.toml", ".ruff.toml", "pyproject.toml"} {
		repoConfig := pocket.FromGitRoot(configName)
		if _, err := os.Stat(repoConfig); err == nil {
			return repoConfig, nil
		}
	}

	// Write bundled config to .pocket/tools/ruff/ruff.toml.
	configDir := pocket.FromToolsDir(name)
	configPath := filepath.Join(configDir, "ruff.toml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			return "", fmt.Errorf("create config dir: %w", err)
		}
		if err := os.WriteFile(configPath, defaultConfig, 0o644); err != nil {
			return "", fmt.Errorf("write default config: %w", err)
		}
	}

	return configPath, nil
}

// Prepare ensures ruff is installed.
var Prepare = tool.PythonToolPreparer(name, version, pythonVersion, uv.CreateVenv, uv.PipInstall)
