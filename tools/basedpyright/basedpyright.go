// Package basedpyright provides basedpyright (Python static type checker) tool integration.
// basedpyright is installed via uv into a virtual environment.
package basedpyright

import (
	"context"
	"os"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// Name is the binary name for basedpyright.
const Name = "basedpyright"

// renovate: datasource=pypi depName=basedpyright
const Version = "1.37.0"

// Install ensures basedpyright is available.
var Install = pocket.Task("install:basedpyright", "install basedpyright", pocket.Serial(
	uv.Install,
	installBasedpyright(),
), pocket.AsHidden())

func installBasedpyright() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		venvDir := pocket.FromToolsDir("basedpyright", Version)
		binary := uv.BinaryPath(venvDir, "basedpyright")

		// Skip if already installed.
		if _, err := os.Stat(binary); err == nil {
			_, err := pocket.CreateSymlink(binary)
			return err
		}

		// Create virtual environment (uv auto-installs if needed).
		// Empty string uses uv.DefaultPythonVersion.
		if err := uv.CreateVenv(ctx, venvDir, ""); err != nil {
			return err
		}

		// Install the package.
		if err := uv.PipInstall(ctx, venvDir, "basedpyright=="+Version); err != nil {
			return err
		}

		// Create symlink to .pocket/bin/.
		_, err := pocket.CreateSymlink(binary)
		return err
	})
}
