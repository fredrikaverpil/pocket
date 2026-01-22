// Package uv provides uv (Python package manager) tool integration.
//
// # Usage Modes
//
// This package supports two modes for running Python tools:
//
// ## Standalone Tools (.pocket/tools/)
//
// For tools managed by pocket (installed once, shared across runs):
//
//	uv.CreateVenv(ctx, ".pocket/tools/ruff/0.14.0", "")
//	uv.PipInstall(ctx, venvDir, "ruff==0.14.0")
//
// ## Project Tools (.pocket/venvs/)
//
// For tools defined in pyproject.toml (project-specific versions):
//
//	uv.Sync(ctx, "3.9", true)
//	uv.Run(ctx, "3.9", "ruff", "check", ".")
package uv

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/fredrikaverpil/pocket/pk"
)

// Name is the binary name for uv.
const Name = "uv"

// Version is the version of uv to install.
// renovate: datasource=github-releases depName=astral-sh/uv
const Version = "0.7.13"

// DefaultPythonVersion is the Python version used when none is specified.
// renovate: datasource=github-releases depName=python/cpython
const DefaultPythonVersion = "3.14.2"

// Install ensures uv is available.
var Install = pk.NewTask("install:uv", "install uv", nil,
	installUV(),
).Hidden()

func installUV() pk.Runnable {
	binDir := pk.FromToolsDir("uv", Version, "bin")
	binaryName := pk.BinaryName("uv")
	binaryPath := filepath.Join(binDir, binaryName)

	url := fmt.Sprintf(
		"https://github.com/astral-sh/uv/releases/download/%s/uv-%s.%s",
		Version,
		platformArch(),
		pk.DefaultArchiveFormat(),
	)

	return pk.Download(url,
		pk.WithDestDir(binDir),
		pk.WithFormat(pk.DefaultArchiveFormat()),
		pk.WithExtract(pk.WithExtractFile(binaryName)),
		pk.WithSymlink(),
		pk.WithSkipIfExists(binaryPath),
	)
}

func platformArch() string {
	switch runtime.GOOS {
	case pk.Darwin:
		if runtime.GOARCH == pk.ARM64 {
			return "aarch64-apple-darwin"
		}
		return "x86_64-apple-darwin"
	case pk.Linux:
		if runtime.GOARCH == pk.ARM64 {
			return "aarch64-unknown-linux-gnu"
		}
		return "x86_64-unknown-linux-gnu"
	case pk.Windows:
		return "x86_64-pc-windows-msvc"
	default:
		return fmt.Sprintf("%s-%s", runtime.GOARCH, runtime.GOOS)
	}
}

// CreateVenv creates a Python virtual environment at the specified path.
// If pythonVersion is empty, DefaultPythonVersion is used.
func CreateVenv(ctx context.Context, venvPath, pythonVersion string) error {
	if pythonVersion == "" {
		pythonVersion = DefaultPythonVersion
	}
	args := []string{"venv", "--python", pythonVersion, venvPath}
	return pk.Exec(ctx, Name, args...)
}

// PipInstall installs a package into a virtual environment.
func PipInstall(ctx context.Context, venvPath, pkg string) error {
	return pk.Exec(ctx, Name, "pip", "install", "--python", venvPython(venvPath), pkg)
}

// PipInstallRequirements installs packages from a requirements.txt file.
func PipInstallRequirements(ctx context.Context, venvPath, requirementsPath string) error {
	return pk.Exec(ctx, Name, "pip", "install", "--python", venvPython(venvPath), "-r", requirementsPath)
}

// venvPython returns the path to the Python executable in a venv.
func venvPython(venvPath string) string {
	if runtime.GOOS == pk.Windows {
		return filepath.Join(venvPath, "Scripts", "python.exe")
	}
	return filepath.Join(venvPath, "bin", "python")
}

// BinaryPath returns the cross-platform path to a binary in a Python venv.
func BinaryPath(venvDir, name string) string {
	if runtime.GOOS == pk.Windows {
		return filepath.Join(venvDir, "Scripts", name+".exe")
	}
	return filepath.Join(venvDir, "bin", name)
}

// ProjectVenvPath returns the path to the project venv for a Python version.
func ProjectVenvPath(pythonVersion string) string {
	if pythonVersion == "" {
		pythonVersion = DefaultPythonVersion
	}
	return pk.FromPocketDir("venvs", pythonVersion)
}

// Sync runs uv sync to install project dependencies into .pocket/venvs/<version>/.
func Sync(ctx context.Context, pythonVersion string, allGroups bool) error {
	if pythonVersion == "" {
		pythonVersion = DefaultPythonVersion
	}

	venvPath := ProjectVenvPath(pythonVersion)
	if pk.Verbose(ctx) {
		pk.Printf(ctx, "Syncing Python %s dependencies to %s\n", pythonVersion, venvPath)
	}

	args := []string{"sync", "--python", pythonVersion}
	if allGroups {
		args = append(args, "--all-groups")
	}

	cmd := exec.CommandContext(ctx, Name, args...)
	cmd.Dir = pk.FromGitRoot(pk.PathFromContext(ctx))
	cmd.Env = append(cmd.Environ(), "UV_PROJECT_ENVIRONMENT="+venvPath)

	out := pk.OutputFromContext(ctx)
	cmd.Stdout = out.Stdout
	cmd.Stderr = out.Stderr

	return cmd.Run()
}

// Run executes a command using uv run from .pocket/venvs/<version>/.
func Run(ctx context.Context, pythonVersion, cmdName string, args ...string) error {
	if pythonVersion == "" {
		pythonVersion = DefaultPythonVersion
	}

	venvPath := ProjectVenvPath(pythonVersion)
	if pk.Verbose(ctx) {
		pk.Printf(ctx, "Running %s from %s\n", cmdName, venvPath)
	}

	uvArgs := []string{"run", "--python", pythonVersion, cmdName}
	uvArgs = append(uvArgs, args...)

	cmd := exec.CommandContext(ctx, Name, uvArgs...)
	cmd.Dir = pk.FromGitRoot(pk.PathFromContext(ctx))
	cmd.Env = append(cmd.Environ(), "UV_PROJECT_ENVIRONMENT="+venvPath)

	out := pk.OutputFromContext(ctx)
	cmd.Stdout = out.Stdout
	cmd.Stderr = out.Stderr

	return cmd.Run()
}
