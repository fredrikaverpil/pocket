// Package mdformat provides mdformat (Markdown formatter) tool integration.
// mdformat is installed via uv into a virtual environment with plugins.
package mdformat

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fredrikaverpil/bld"
	"github.com/fredrikaverpil/bld/tools/uv"
)

const name = "mdformat"

// Python 3.13+ required by mdformat v1 for --exclude support.
const pythonVersion = "3.13"

//go:embed requirements.txt
var requirements []byte

// Command prepares the tool and returns an exec.Cmd for running mdformat.
func Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
	if err := Prepare(ctx); err != nil {
		return nil, err
	}
	return bld.Command(ctx, bld.FromBinDir(name), args...), nil
}

// Run installs (if needed) and executes mdformat.
func Run(ctx context.Context, args ...string) error {
	cmd, err := Command(ctx, args...)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// versionHash creates a unique hash based on requirements and Python version.
// This ensures the venv is recreated when dependencies or Python version change.
func versionHash() string {
	h := sha256.New()
	h.Write(requirements)
	h.Write([]byte(pythonVersion))
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// Prepare ensures mdformat is installed.
func Prepare(ctx context.Context) error {
	// Use hash-based versioning: .bld/tools/mdformat/<hash>/
	venvDir := bld.FromToolsDir(name, versionHash())
	binary := filepath.Join(venvDir, "bin", name)

	// Skip if already installed
	if _, err := os.Stat(binary); err == nil {
		return nil
	}

	// Create virtual environment with Python 3.13+ for --exclude support
	if err := uv.CreateVenv(ctx, venvDir, pythonVersion); err != nil {
		return err
	}

	// Write requirements.txt to venv dir
	reqPath := filepath.Join(venvDir, "requirements.txt")
	if err := os.WriteFile(reqPath, requirements, 0o644); err != nil {
		return err
	}

	// Install from requirements.txt
	if err := uv.PipInstallRequirements(ctx, venvDir, reqPath); err != nil {
		return err
	}

	// Create symlink to .bld/bin/
	binDir := bld.FromBinDir()
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	symlinkPath := filepath.Join(binDir, name)
	_ = os.Remove(symlinkPath) // Remove existing symlink if any
	return os.Symlink(binary, symlinkPath)
}
