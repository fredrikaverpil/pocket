// Package basedpyright provides basedpyright (Python static type checker) tool integration.
// basedpyright is installed via uv into a virtual environment.
package basedpyright

import (
	"context"
	"os"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

const name = "basedpyright"

// renovate: datasource=pypi depName=basedpyright
const version = "1.37.0"

// pythonVersion specifies the Python version for the virtual environment.
const pythonVersion = "3.12"

// Tool is the basedpyright tool.
//
// Example usage in a task action:
//
//	basedpyright.Tool.Exec(ctx, tc, ".")
var Tool = pocket.NewTool(name, version, install)

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
