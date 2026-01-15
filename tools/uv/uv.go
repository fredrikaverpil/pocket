// Package uv provides uv (Python package manager) tool integration.
package uv

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/fredrikaverpil/pocket"
)

// Name is the binary name for uv.
const Name = "uv"

// renovate: datasource=github-releases depName=astral-sh/uv
const Version = "0.7.13"

// Install ensures uv is available.
var Install = pocket.Func("install:uv", "install uv", install).Hidden()

func install(ctx context.Context) error {
	binDir := pocket.FromToolsDir("uv", Version, "bin")
	binaryName := pocket.BinaryName("uv")
	binaryPath := filepath.Join(binDir, binaryName)

	url := fmt.Sprintf(
		"https://github.com/astral-sh/uv/releases/download/%s/uv-%s.%s",
		Version,
		platformArch(),
		pocket.DefaultArchiveFormat(),
	)

	return pocket.Download(ctx, url,
		pocket.WithDestDir(binDir),
		pocket.WithFormat(pocket.DefaultArchiveFormat()),
		pocket.WithExtract(pocket.WithExtractFile(binaryName)),
		pocket.WithSymlink(),
		pocket.WithSkipIfExists(binaryPath),
	)
}

// CreateVenv creates a Python virtual environment at the specified path.
// If pythonVersion is empty, uv uses the default Python available.
// NOTE: Callers must ensure uv.Install has been composed as a dependency.
func CreateVenv(ctx context.Context, venvPath, pythonVersion string) error {
	args := []string{"venv"}
	if pythonVersion != "" {
		args = append(args, "--python", pythonVersion)
	}
	args = append(args, venvPath)
	return pocket.Exec(ctx, Name, args...)
}

// PipInstall installs a package into a virtual environment.
// NOTE: Callers must ensure uv.Install has been composed as a dependency.
func PipInstall(ctx context.Context, venvPath, pkg string) error {
	return pocket.Exec(ctx, Name, "pip", "install", "--python", venvPython(venvPath), pkg)
}

// PipInstallRequirements installs packages from a requirements.txt file.
// NOTE: Callers must ensure uv.Install has been composed as a dependency.
func PipInstallRequirements(ctx context.Context, venvPath, requirementsPath string) error {
	return pocket.Exec(ctx, Name, "pip", "install", "--python", venvPython(venvPath), "-r", requirementsPath)
}

// venvPython returns the path to the Python executable in a venv.
// On Windows, it's Scripts\python.exe; on Unix, it's bin/python.
func venvPython(venvPath string) string {
	if runtime.GOOS == pocket.Windows {
		return filepath.Join(venvPath, "Scripts", "python.exe")
	}
	return filepath.Join(venvPath, "bin", "python")
}

func platformArch() string {
	switch runtime.GOOS {
	case pocket.Darwin:
		if runtime.GOARCH == pocket.ARM64 {
			return "aarch64-apple-darwin"
		}
		return "x86_64-apple-darwin"
	case pocket.Linux:
		if runtime.GOARCH == pocket.ARM64 {
			return "aarch64-unknown-linux-gnu"
		}
		return "x86_64-unknown-linux-gnu"
	case pocket.Windows:
		return "x86_64-pc-windows-msvc"
	default:
		return fmt.Sprintf("%s-%s", runtime.GOARCH, runtime.GOOS)
	}
}
