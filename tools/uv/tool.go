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
	"github.com/goyek/goyek/v3"
)

const name = "uv"

// renovate: datasource=github-releases depName=astral-sh/uv
const version = "0.7.13"

// Prepare is a goyek task that downloads and installs uv.
// Hidden from task list (no Usage field).
var Prepare = goyek.Define(goyek.Task{
	Name: "uv:prepare",
	Action: func(a *goyek.A) {
		if err := prepare(a.Context()); err != nil {
			a.Fatal(err)
		}
	},
})

// Command returns an exec.Cmd for running uv.
// Call Prepare first or use as a goyek Deps.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	return bld.Command(ctx, bld.FromBinDir(name), args...)
}

// Run executes uv with the given arguments.
// Call Prepare first or use as a goyek Deps.
func Run(ctx context.Context, args ...string) error {
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

func prepare(ctx context.Context) error {
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
