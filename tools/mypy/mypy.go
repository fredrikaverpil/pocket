// Package mypy provides mypy (Python static type checker) tool integration.
// mypy is installed via uv into a virtual environment.
package mypy

import (
	"context"
	"os"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// Name is the binary name for mypy.
const Name = "mypy"

// renovate: datasource=pypi depName=mypy
const Version = "1.19.1"

// pythonVersion specifies the Python version for the virtual environment.
const pythonVersion = "3.12"

// Install ensures mypy is available.
var Install = pocket.Task("install:mypy", "install mypy", pocket.Serial(
	uv.Install,
	installMypy(),
)).Hidden()

func installMypy() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		venvDir := pocket.FromToolsDir("mypy", Version)
		binary := uv.BinaryPath(venvDir, "mypy")

		// Skip if already installed.
		if _, err := os.Stat(binary); err == nil {
			_, err := pocket.CreateSymlink(binary)
			return err
		}

		// Create virtual environment (uv auto-installs if needed).
		if err := uv.CreateVenv(ctx, venvDir, pythonVersion); err != nil {
			return err
		}

		// Install the package.
		if err := uv.PipInstall(ctx, venvDir, "mypy=="+Version); err != nil {
			return err
		}

		// Create symlink to .pocket/bin/.
		_, err := pocket.CreateSymlink(binary)
		return err
	})
}
