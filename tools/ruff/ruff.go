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

// Name is the binary name for ruff.
const Name = "ruff"

// renovate: datasource=pypi depName=ruff
const Version = "0.14.0"

// pythonVersion specifies the Python version for the virtual environment.
const pythonVersion = "3.12"

//go:embed ruff.toml
var defaultConfig []byte

// Config describes how to find or create ruff's configuration file.
var Config = pocket.ToolConfig{
	UserFiles:   []string{"ruff.toml", ".ruff.toml", "pyproject.toml"},
	DefaultFile: "ruff.toml",
	DefaultData: defaultConfig,
}

// Install ensures ruff is available.
var Install = pocket.Func("install:ruff", "install ruff", install).Hidden()

func install(ctx context.Context) error {
	venvDir := pocket.FromToolsDir("ruff", Version)
	binary := pocket.VenvBinaryPath(venvDir, "ruff")

	// Skip if already installed.
	if _, err := os.Stat(binary); err == nil {
		_, err := pocket.CreateSymlink(binary)
		return err
	}

	pocket.Printf(ctx, "Installing ruff %s...\n", Version)

	// Create virtual environment (uv auto-installs if needed).
	if err := uv.CreateVenv(ctx, venvDir, pythonVersion); err != nil {
		return err
	}

	// Install the package.
	if err := uv.PipInstall(ctx, venvDir, "ruff=="+Version); err != nil {
		return err
	}

	// Create symlink to .pocket/bin/.
	_, err := pocket.CreateSymlink(binary)
	return err
}
