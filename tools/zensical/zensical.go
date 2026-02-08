// Package zensical provides the zensical tool for documentation generation.
package zensical

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// Name is the binary name for zensical.
const Name = "zensical"

//go:embed pyproject.toml
var pyprojectTOML []byte

//go:embed uv.lock
var uvLock []byte

// Version returns a content hash based on pyproject.toml, uv.lock, and Python version.
func Version() string {
	return uv.ContentHash(pyprojectTOML, uvLock, []byte(uv.DefaultPythonVersion))
}

// Install is a hidden, global task that installs zensical.
// Global ensures it only runs once regardless of path context.
var Install = &pk.Task{
	Name:   "install:zensical",
	Usage:  "install zensical",
	Body:   pk.Serial(uv.Install, installZensical()),
	Hidden: true,
	Global: true,
}

func installZensical() pk.Runnable {
	installDir := pk.FromToolsDir(Name, Version())
	venvPath := filepath.Join(installDir, "venv")
	return uv.EnsureInstalled(venvPath, Name, func(ctx context.Context) error {
		// Create install directory and write project files.
		if err := os.MkdirAll(installDir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(installDir, "pyproject.toml"), pyprojectTOML, 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(installDir, "uv.lock"), uvLock, 0o644); err != nil {
			return err
		}

		// Sync dependencies using uv.
		return uv.Sync(ctx, uv.SyncOptions{
			PythonVersion: uv.DefaultPythonVersion,
			VenvPath:      venvPath,
			ProjectDir:    installDir,
		})
	})
}

// Exec runs zensical with the given arguments.
// The working directory is determined by the context path.
func Exec(ctx context.Context, args ...string) error {
	installDir := pk.FromToolsDir(Name, Version())
	venvDir := filepath.Join(installDir, "venv")
	return uv.ExecTool(ctx, venvDir, Name, args...)
}
