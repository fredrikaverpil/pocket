// Package ruff provides ruff (Python linter and formatter) tool integration.
// ruff is installed via uv into a virtual environment.
package ruff

import (
	"context"
	_ "embed"
	"os"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

const name = "ruff"

// renovate: datasource=pypi depName=ruff
const version = "0.14.0"

// pythonVersion specifies the Python version for the virtual environment.
const pythonVersion = "3.12"

//go:embed ruff.toml
var defaultConfig []byte

// Tool is the ruff tool.
//
// Example usage in a task action:
//
//	configPath, _ := ruff.Tool.ConfigPath()
//	ruff.Tool.Exec(ctx, tc, "check", "--config", configPath, ".")
var Tool = pocket.NewTool(name, version, install).
	WithConfig(pocket.ToolConfig{
		UserFiles:   []string{"ruff.toml", ".ruff.toml", "pyproject.toml"},
		DefaultFile: "ruff.toml",
		DefaultData: defaultConfig,
	})

func install(ctx context.Context, tc *pocket.TaskContext) error {
	venvDir := pocket.FromToolsDir(name, version)
	binary := pocket.VenvBinaryPath(venvDir, name)

	// Skip if already installed.
	if _, err := os.Stat(binary); err == nil {
		_, err := pocket.CreateSymlink(binary)
		return err
	}

	// Create virtual environment (uv auto-installs if needed).
	if err := uv.CreateVenv(ctx, tc, venvDir, pythonVersion); err != nil {
		return err
	}

	// Install the package.
	if err := uv.PipInstall(ctx, tc, venvDir, name+"=="+version); err != nil {
		return err
	}

	// Create symlink to .pocket/bin/.
	_, err := pocket.CreateSymlink(binary)
	return err
}
