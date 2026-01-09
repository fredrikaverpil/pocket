package tool

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fredrikaverpil/pocket"
)

// VenvBinaryPath returns the path to a binary in a Python virtual environment.
// On Windows, binaries are in Scripts/ with .exe extension.
// On Unix, binaries are in bin/ without extension.
func VenvBinaryPath(venvDir, name string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", name+".exe")
	}
	return filepath.Join(venvDir, "bin", name)
}

// PythonToolPreparer creates a Prepare function for Python tools installed via uv.
// This handles the common pattern of:
// 1. Check if binary exists, return early if so (with symlink)
// 2. Create virtual environment with specified Python version
// 3. Install the package
// 4. Create symlink to .pocket/bin/
//
// createVenv and pipInstall are passed as functions to avoid import cycles
// (this package cannot import tools/uv).
func PythonToolPreparer(
	name, version, pythonVersion string,
	createVenv func(ctx context.Context, venvPath, pyVersion string) error,
	pipInstall func(ctx context.Context, venvPath, pkg string) error,
) func(context.Context) error {
	return func(ctx context.Context) error {
		venvDir := pocket.FromToolsDir(name, version)
		binary := VenvBinaryPath(venvDir, name)

		// Skip if already installed.
		if _, err := os.Stat(binary); err == nil {
			_, err := CreateSymlink(binary)
			return err
		}

		// Create virtual environment.
		if err := createVenv(ctx, venvDir, pythonVersion); err != nil {
			return err
		}

		// Install the package.
		if err := pipInstall(ctx, venvDir, name+"=="+version); err != nil {
			return err
		}

		// Create symlink to .pocket/bin/.
		_, err := CreateSymlink(binary)
		return err
	}
}
