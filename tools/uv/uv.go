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
//	uv.Sync(ctx, uv.SyncOptions{PythonVersion: "3.9", AllGroups: true})
//	uv.Run(ctx, uv.RunOptions{PythonVersion: "3.9"}, "ruff", "check", ".")
//
// Venvs are created at .pocket/venvs/<project-path>/venv-<version>/ by default,
// where <project-path> is derived from PathFromContext. This ensures multiple
// projects in a monorepo don't collide.
package uv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/download"
)

// Name is the binary name for uv.
const Name = "uv"

// Version is the version of uv to install.
// renovate: datasource=github-releases depName=astral-sh/uv
const Version = "0.10.0"

// DefaultPythonVersion is the Python version used when none is specified.
// renovate: datasource=github-releases depName=python/cpython
const DefaultPythonVersion = "3.14.2"

// Install ensures uv is available.
var Install = &pk.Task{
	Name:   "install:uv",
	Usage:  "install uv",
	Body:   installUV(),
	Hidden: true,
	Global: true,
}

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

	return download.Download(url,
		download.WithDestDir(binDir),
		download.WithFormat(pk.DefaultArchiveFormat()),
		download.WithExtract(download.WithExtractFile(binaryName)),
		download.WithSymlink(),
		download.WithSkipIfExists(binaryPath),
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

// venvPython returns the path to the Python executable in a venv.
func venvPython(venvPath string) string {
	if runtime.GOOS == pk.Windows {
		return filepath.Join(venvPath, "Scripts", "python.exe")
	}
	return filepath.Join(venvPath, "bin", "python")
}

// IsInstalled reports whether a Python tool is properly installed in a venv.
// It checks that both the tool binary and the venv's Python interpreter exist.
// This guards against stale caches where script files remain but the Python
// interpreter referenced by their shebang is missing.
func IsInstalled(venvDir, name string) bool {
	if _, err := os.Stat(BinaryPath(venvDir, name)); err != nil {
		return false
	}
	if _, err := os.Stat(venvPython(venvDir)); err != nil {
		return false
	}
	return true
}

// EnsureInstalled returns a Runnable that skips installFn if the tool is already
// properly installed (binary + Python interpreter both exist). Use this instead
// of calling IsInstalled manually to ensure the check is never forgotten.
func EnsureInstalled(venvDir, name string, installFn func(ctx context.Context) error) pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		if IsInstalled(venvDir, name) {
			return nil
		}
		return installFn(ctx)
	})
}

// BinaryPath returns the cross-platform path to a binary in a Python venv.
func BinaryPath(venvDir, name string) string {
	if runtime.GOOS == pk.Windows {
		return filepath.Join(venvDir, "Scripts", name+".exe")
	}
	return filepath.Join(venvDir, "bin", name)
}

// ContentHash returns a short hash for use as a directory name.
// Pass all embedded files + python version to get a content-addressable key.
func ContentHash(data ...[]byte) string {
	h := sha256.New()
	for _, d := range data {
		h.Write(d)
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// ExecTool runs a Python tool by invoking the venv's Python interpreter
// with the script path. This avoids shebang dependencies.
func ExecTool(ctx context.Context, venvDir, name string, args ...string) error {
	python := venvPython(venvDir)
	script := BinaryPath(venvDir, name)
	execArgs := append([]string{script}, args...)
	return pk.Exec(ctx, python, execArgs...)
}

// DefaultVenvPattern is the naming pattern for venvs. %s is replaced with the Python version.
const DefaultVenvPattern = "venv-%s"

// SyncOptions configures uv sync behavior.
type SyncOptions struct {
	// PythonVersion specifies which Python version to use.
	// If empty, DefaultPythonVersion is used.
	PythonVersion string

	// VenvPath is the explicit path to the venv directory.
	// If empty, it's computed from ProjectDir and PythonVersion.
	VenvPath string

	// ProjectDir is where pyproject.toml lives.
	// If empty, PathFromContext(ctx) is used.
	ProjectDir string

	// AllGroups installs all dependency groups from pyproject.toml.
	AllGroups bool
}

// RunOptions configures uv run behavior.
type RunOptions struct {
	// PythonVersion specifies which Python version to use.
	// If empty, DefaultPythonVersion is used.
	PythonVersion string

	// VenvPath is the explicit path to the venv directory.
	// If empty, it's computed from ProjectDir and PythonVersion.
	VenvPath string

	// ProjectDir is where pyproject.toml lives.
	// If empty, PathFromContext(ctx) is used.
	ProjectDir string
}

// VenvPath computes the venv path for a project.
// If projectPath is empty or ".", returns .pocket/venvs/venv-<version>/.
// Otherwise returns .pocket/venvs/<projectPath>/venv-<version>/.
func VenvPath(projectPath, pythonVersion string) string {
	if pythonVersion == "" {
		pythonVersion = DefaultPythonVersion
	}
	venvName := fmt.Sprintf(DefaultVenvPattern, pythonVersion)
	if projectPath == "" || projectPath == "." {
		return pk.FromPocketDir("venvs", venvName)
	}
	return pk.FromPocketDir("venvs", projectPath, venvName)
}

// Sync runs uv sync to install project dependencies.
// Venv is created at .pocket/venvs/<project-path>/venv-<version>/ by default.
func Sync(ctx context.Context, opts SyncOptions) error {
	pythonVersion := opts.PythonVersion
	if pythonVersion == "" {
		pythonVersion = DefaultPythonVersion
	}

	projectDir := opts.ProjectDir
	if projectDir == "" {
		projectDir = pk.PathFromContext(ctx)
	}

	venvPath := opts.VenvPath
	if venvPath == "" {
		venvPath = VenvPath(projectDir, pythonVersion)
	}

	if pk.Verbose(ctx) {
		pk.Printf(ctx, "Syncing Python %s dependencies to %s\n", pythonVersion, venvPath)
	}

	args := []string{"sync", "--python", pythonVersion}
	if opts.AllGroups {
		args = append(args, "--all-groups")
	}

	ctx = pk.ContextWithPath(ctx, projectDir)
	ctx = pk.ContextWithoutEnv(ctx, "VIRTUAL_ENV")
	ctx = pk.ContextWithEnv(ctx, "UV_PROJECT_ENVIRONMENT="+venvPath)

	return pk.Exec(ctx, Name, args...)
}

// Run executes a command using uv run.
// Uses venv at .pocket/venvs/<project-path>/venv-<version>/ by default.
func Run(ctx context.Context, opts RunOptions, cmdName string, args ...string) error {
	pythonVersion := opts.PythonVersion
	if pythonVersion == "" {
		pythonVersion = DefaultPythonVersion
	}

	projectDir := opts.ProjectDir
	if projectDir == "" {
		projectDir = pk.PathFromContext(ctx)
	}

	venvPath := opts.VenvPath
	if venvPath == "" {
		venvPath = VenvPath(projectDir, pythonVersion)
	}

	if pk.Verbose(ctx) {
		pk.Printf(ctx, "Running %s from %s\n", cmdName, venvPath)
	}

	uvArgs := []string{"run", "--python", pythonVersion, cmdName}
	uvArgs = append(uvArgs, args...)

	ctx = pk.ContextWithPath(ctx, projectDir)
	ctx = pk.ContextWithoutEnv(ctx, "VIRTUAL_ENV")
	ctx = pk.ContextWithEnv(ctx, "UV_PROJECT_ENVIRONMENT="+venvPath)

	return pk.Exec(ctx, Name, uvArgs...)
}
