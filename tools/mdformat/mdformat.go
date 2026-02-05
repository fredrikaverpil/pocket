// Package mdformat provides mdformat (Markdown formatter) tool integration.
// mdformat is installed via uv into a virtual environment with plugins.
package mdformat

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// Name is the binary name for mdformat.
const Name = "mdformat"

// Python 3.13+ required by mdformat v1 for --exclude support.
const pythonVersion = "3.13"

//go:embed requirements.txt
var requirements []byte

// Version creates a unique hash based on requirements and Python version.
func Version() string {
	h := sha256.New()
	h.Write(requirements)
	h.Write([]byte(pythonVersion))
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// Install ensures mdformat is available.
var Install = &pk.Task{
	Name:   "install:mdformat",
	Usage:  "install mdformat",
	Body:   pk.Serial(uv.Install, installMdformat()),
	Hidden: true,
	Global: true,
}

func installMdformat() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		venvDir := pk.FromToolsDir("mdformat", Version())
		binary := uv.BinaryPath(venvDir, "mdformat")

		// Skip if already installed.
		if _, err := os.Stat(binary); err == nil {
			return nil
		}

		// Create virtual environment with Python 3.13+.
		if err := uv.CreateVenv(ctx, venvDir, pythonVersion); err != nil {
			return err
		}

		// Write requirements.txt to venv dir.
		reqPath := filepath.Join(venvDir, "requirements.txt")
		if err := os.WriteFile(reqPath, requirements, 0o644); err != nil {
			return err
		}

		// Install from requirements.txt.
		// No symlink needed since Exec() runs via venv Python.
		return uv.PipInstallRequirements(ctx, venvDir, reqPath)
	})
}

// Exec runs mdformat with the given arguments.
func Exec(ctx context.Context, args ...string) error {
	venvDir := pk.FromToolsDir("mdformat", Version())
	python := uv.BinaryPath(venvDir, "python")
	// Run as module to avoid shebang path issues.
	execArgs := append([]string{"-m", "mdformat"}, args...)
	return pk.Exec(ctx, python, execArgs...)
}
