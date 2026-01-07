// Package uv provides uv (Python package manager) tool integration.
package uv

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tool"
)

const name = "uv"

// renovate: datasource=github-releases depName=astral-sh/uv
const version = "0.7.13"

// Command prepares the tool and returns an exec.Cmd for running uv.
func Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
	if err := Prepare(ctx); err != nil {
		return nil, err
	}
	return pocket.Command(ctx, pocket.FromBinDir(pocket.BinaryName(name)), args...), nil
}

// Run installs (if needed) and executes uv.
func Run(ctx context.Context, args ...string) error {
	cmd, err := Command(ctx, args...)
	if err != nil {
		return err
	}
	return cmd.Run()
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
	return Run(ctx, "pip", "install", "--python", venvPython(venvPath), pkg)
}

// PipInstallRequirements installs packages from a requirements.txt file.
func PipInstallRequirements(ctx context.Context, venvPath, requirementsPath string) error {
	return Run(ctx, "pip", "install", "--python", venvPython(venvPath), "-r", requirementsPath)
}

// venvPython returns the path to the Python executable in a venv.
// On Windows, it's Scripts\python.exe; on Unix, it's bin/python.
func venvPython(venvPath string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Scripts", "python.exe")
	}
	return filepath.Join(venvPath, "bin", "python")
}

// Prepare ensures uv is installed.
func Prepare(ctx context.Context) error {
	binDir := pocket.FromToolsDir(name, version, "bin")
	binaryName := name
	if runtime.GOOS == "windows" {
		binaryName = name + ".exe"
	}
	binary := filepath.Join(binDir, binaryName)

	// Windows uses .zip, others use .tar.gz.
	var binURL string
	var opts []tool.Opt
	if runtime.GOOS == "windows" {
		binURL = fmt.Sprintf(
			"https://github.com/astral-sh/uv/releases/download/%s/uv-%s.zip",
			version,
			platformArch(),
		)
		opts = []tool.Opt{
			tool.WithDestinationDir(binDir),
			tool.WithUnzip(),
			tool.WithExtractFiles(binaryName),
			tool.WithSkipIfFileExists(binary),
			tool.WithSymlink(binary),
		}
	} else {
		binURL = fmt.Sprintf(
			"https://github.com/astral-sh/uv/releases/download/%s/uv-%s.tar.gz",
			version,
			platformArch(),
		)
		opts = []tool.Opt{
			tool.WithDestinationDir(binDir),
			tool.WithUntarGz(),
			tool.WithExtractFiles(binaryName),
			tool.WithSkipIfFileExists(binary),
			tool.WithSymlink(binary),
		}
	}

	return tool.FromRemote(ctx, binURL, opts...)
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
