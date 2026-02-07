// Package zensical provides the zensical tool for documentation generation.
package zensical

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// Name is the binary name for zensical.
const Name = "zensical"

//go:embed pyproject.toml
var pyprojectToml []byte

//go:embed uv.lock
var uvLock []byte

var (
	versionOnce sync.Once
	version     string
)

// Version returns the zensical version from pyproject.toml.
// renovate: datasource=pypi depName=zensical
func Version() string {
	versionOnce.Do(func() {
		// Parse version from dependencies array in pyproject.toml
		// Looking for: "zensical==X.Y.Z"
		re := regexp.MustCompile(`"zensical==([^"]+)"`)
		if matches := re.FindSubmatch(pyprojectToml); len(matches) > 1 {
			version = string(matches[1])
		}
	})
	return version
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
		binary := uv.BinaryPath(venvPath, Name)

		// Skip if already installed.
		if _, err := os.Stat(binary); err == nil {
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

// InstallDir returns the installation directory for zensical.
func InstallDir() string {
	return pk.FromToolsDir(Name, Version())
}

// VenvPath returns the virtual environment path for zensical.
func VenvPath() string {
	return filepath.Join(InstallDir(), "venv")
}
