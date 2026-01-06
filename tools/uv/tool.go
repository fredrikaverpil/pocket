// Package uv provides uv (Python package manager) tool integration.
package uv

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/fredrikaverpil/bld"
	"github.com/fredrikaverpil/bld/tool"
)

const name = "uv"

// renovate: datasource=github-releases depName=astral-sh/uv
const version = "0.7.13"

// Command returns an exec.Cmd for running uv.
// Prefer Run() which auto-prepares the tool.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	return bld.Command(ctx, bld.FromBinDir(name), args...)
}

// Run installs (if needed) and executes uv.
func Run(ctx context.Context, args ...string) error {
	if err := Prepare(ctx); err != nil {
		return err
	}
	return Command(ctx, args...).Run()
}

// CreateVenv creates a Python virtual environment at the specified path.
// If pythonVersion is empty, uv uses the default Python available.
func CreateVenv(ctx context.Context, venvPath, pythonVersion string) error {
	args := []string{"venv"}
	if pythonVersion != "" {
		args = append(args, "--python", pythonVersion)
	}
	args = append(args, venvPath)
	return Run(ctx, args...)
}

// PipInstall installs a package into a virtual environment.
func PipInstall(ctx context.Context, venvPath, pkg string) error {
	return Run(ctx, "pip", "install", "--python", filepath.Join(venvPath, "bin", "python"), pkg)
}

// PipInstallRequirements installs packages from a requirements.txt file.
func PipInstallRequirements(ctx context.Context, venvPath, requirementsPath string) error {
	return Run(ctx, "pip", "install", "--python", filepath.Join(venvPath, "bin", "python"), "-r", requirementsPath)
}

// Prepare ensures uv is installed.
func Prepare(ctx context.Context) error {
	binDir := bld.FromToolsDir(name, version, "bin")
	binary := filepath.Join(binDir, name)

	binURL := fmt.Sprintf(
		"https://github.com/astral-sh/uv/releases/download/%s/uv-%s.tar.gz",
		version,
		platformArch(),
	)

	return tool.FromRemote(
		ctx,
		binURL,
		tool.WithDestinationDir(binDir),
		tool.WithUntarGz(),
		tool.WithExtractFiles(name),
		tool.WithSkipIfFileExists(binary),
		tool.WithSymlink(binary),
	)
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
