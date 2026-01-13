// Package uv provides uv (Python package manager) tool integration.
package uv

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/fredrikaverpil/pocket"
)

const name = "uv"

// renovate: datasource=github-releases depName=astral-sh/uv
const version = "0.7.13"

// Tool is the uv tool.
//
// Example usage in a task action:
//
//	uv.Tool.Exec(ctx, tc, "venv", ".")
var Tool = pocket.NewTool(name, version, install)

// CreateVenv creates a Python virtual environment at the specified path.
// If pythonVersion is empty, uv uses the default Python available.
func CreateVenv(ctx context.Context, tc *pocket.TaskContext, venvPath, pythonVersion string) error {
	args := []string{"venv"}
	if pythonVersion != "" {
		args = append(args, "--python", pythonVersion)
	}
	args = append(args, venvPath)
	return Tool.Exec(ctx, tc, args...)
}

// PipInstall installs a package into a virtual environment.
func PipInstall(ctx context.Context, tc *pocket.TaskContext, venvPath, pkg string) error {
	return Tool.Exec(ctx, tc, "pip", "install", "--python", venvPython(venvPath), pkg)
}

// PipInstallRequirements installs packages from a requirements.txt file.
func PipInstallRequirements(ctx context.Context, tc *pocket.TaskContext, venvPath, requirementsPath string) error {
	return Tool.Exec(ctx, tc, "pip", "install", "--python", venvPython(venvPath), "-r", requirementsPath)
}

// venvPython returns the path to the Python executable in a venv.
// On Windows, it's Scripts\python.exe; on Unix, it's bin/python.
func venvPython(venvPath string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Scripts", "python.exe")
	}
	return filepath.Join(venvPath, "bin", "python")
}

func install(ctx context.Context, tc *pocket.TaskContext) error {
	binDir := pocket.FromToolsDir(name, version, "bin")
	binaryName := pocket.BinaryName(name)

	var format string
	if runtime.GOOS == "windows" {
		format = "zip"
	} else {
		format = "tar.gz"
	}

	url := fmt.Sprintf(
		"https://github.com/astral-sh/uv/releases/download/%s/uv-%s.%s",
		version,
		platformArch(),
		format,
	)

	return pocket.DownloadBinary(ctx, tc, url, pocket.DownloadOpts{
		DestDir:      binDir,
		Format:       format,
		ExtractFiles: []string{binaryName},
		Symlink:      true,
	})
}

func platformArch() string {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "aarch64-apple-darwin"
		}
		return "x86_64-apple-darwin"
	case "linux":
		if runtime.GOARCH == "arm64" {
			return "aarch64-unknown-linux-gnu"
		}
		return "x86_64-unknown-linux-gnu"
	case "windows":
		return "x86_64-pc-windows-msvc"
	default:
		return fmt.Sprintf("%s-%s", runtime.GOARCH, runtime.GOOS)
	}
}
