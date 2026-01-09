// Package ruff provides ruff (Python linter and formatter) tool integration.
// ruff is installed via uv into a virtual environment.
package ruff

import (
	"context"
	_ "embed"
	"os"

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

var configSpec = tool.ConfigSpec{
	ToolName:          name,
	UserConfigNames:   []string{"ruff.toml", ".ruff.toml", "pyproject.toml"},
	DefaultConfigName: "ruff.toml",
	DefaultConfig:     defaultConfig,
}

// ConfigPath returns the path to the ruff config file.
// It checks for ruff.toml, .ruff.toml, or pyproject.toml in the repo root first,
// then falls back to the bundled default config.
var ConfigPath = configSpec.Path

// Prepare ensures ruff is installed.
func Prepare(ctx context.Context) error {
	venvDir := pocket.FromToolsDir(name, version)
	binary := tool.VenvBinaryPath(venvDir, name)

	// Skip if already installed.
	if _, err := os.Stat(binary); err == nil {
		_, err := tool.CreateSymlink(binary)
		return err
	}

	// Create virtual environment.
	if err := uv.CreateVenv(ctx, venvDir, pythonVersion); err != nil {
		return err
	}

	// Install the package.
	if err := uv.PipInstall(ctx, venvDir, name+"=="+version); err != nil {
		return err
	}

	// Create symlink to .pocket/bin/.
	_, err := tool.CreateSymlink(binary)
	return err
}
