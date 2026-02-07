// Package zensical provides zensical (Python documentation tool) integration.
// zensical is installed via uv into a local directory with locked dependencies.
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

// Install ensures zensical is available.
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

// Exec runs zensical with the given arguments.
func Exec(ctx context.Context, args ...string) error {
	installDir := pk.FromToolsDir(Name, Version())
	venvPath := filepath.Join(installDir, "venv")

	return uv.Run(ctx, uv.RunOptions{
		PythonVersion: uv.DefaultPythonVersion,
		VenvPath:      venvPath,
		ProjectDir:    installDir,
	}, Name, args...)
}

// Build runs zensical build command to generate documentation.
func Build(ctx context.Context, args ...string) error {
	buildArgs := append([]string{"build"}, args...)
	return Exec(ctx, buildArgs...)
}

// Serve runs zensical serve command to serve documentation locally.
func Serve(ctx context.Context, args ...string) error {
	serveArgs := append([]string{"serve"}, args...)
	return Exec(ctx, serveArgs...)
}
