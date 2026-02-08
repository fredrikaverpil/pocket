// Package zensical provides the zensical tool for documentation generation.
package zensical

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"regexp"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// Name is the binary name for zensical.
const Name = "zensical"

//go:embed pyproject.toml
var pyprojectToml []byte

//go:embed uv.lock
var uvLock []byte

// Version returns the zensical version from pyproject.toml.
// renovate: datasource=pypi depName=zensical
func Version() string {
	// Parse version from dependencies array in pyproject.toml
	// Looking for: "zensical==X.Y.Z"
	re := regexp.MustCompile(`"zensical==([^"]+)"`)
	if matches := re.FindSubmatch(pyprojectToml); len(matches) > 1 {
		return string(matches[1])
	}
	return ""
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
	return pk.Do(func(ctx context.Context) error {
		installDir := pk.FromToolsDir(Name, Version())
		venvPath := filepath.Join(installDir, "venv")

		// Skip if already installed.
		if uv.IsInstalled(venvPath, Name) {
			return nil
		}

		// Create install directory and write project files.
		if err := os.MkdirAll(installDir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(installDir, "pyproject.toml"), pyprojectToml, 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(installDir, "uv.lock"), uvLock, 0o644); err != nil {
			return err
		}

		// Sync dependencies using uv.
		// Set explicit venv path and project directory.
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
	binary := uv.BinaryPath(VenvPath(), Name)
	return pk.Exec(ctx, binary, args...)
}

// InstallDir returns the installation directory for zensical.
func InstallDir() string {
	return pk.FromToolsDir(Name, Version())
}

// VenvPath returns the virtual environment path for zensical.
func VenvPath() string {
	return filepath.Join(InstallDir(), "venv")
}
