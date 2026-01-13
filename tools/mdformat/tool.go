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

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

const name = "mdformat"

// Python 3.13+ required by mdformat v1 for --exclude support.
const pythonVersion = "3.13"

//go:embed requirements.txt
var requirements []byte

// Tool is the mdformat tool.
//
// Example usage in a task action:
//
//	mdformat.Tool.Exec(ctx, tc, "--wrap", "80", ".")
var Tool = pocket.NewTool(name, versionHash(), install)

// versionHash creates a unique hash based on requirements and Python version.
// This ensures the venv is recreated when dependencies or Python version change.
func versionHash() string {
	h := sha256.New()
	h.Write(requirements)
	h.Write([]byte(pythonVersion))
	return hex.EncodeToString(h.Sum(nil))[:12]
}

func install(ctx context.Context, tc *pocket.TaskContext) error {
	// Use hash-based versioning: .pocket/tools/mdformat/<hash>/
	venvDir := pocket.FromToolsDir(name, versionHash())
	binary := pocket.VenvBinaryPath(venvDir, name)

	// Skip if already installed.
	if _, err := os.Stat(binary); err == nil {
		// Ensure symlink/copy exists.
		_, err := pocket.CreateSymlink(binary)
		return err
	}

	// Create virtual environment with Python 3.13+ for --exclude support.
	if err := uv.CreateVenv(ctx, tc, venvDir, pythonVersion); err != nil {
		return err
	}

	// Write requirements.txt to venv dir.
	reqPath := filepath.Join(venvDir, "requirements.txt")
	if err := os.WriteFile(reqPath, requirements, 0o644); err != nil {
		return err
	}

	// Install from requirements.txt.
	if err := uv.PipInstallRequirements(ctx, tc, venvDir, reqPath); err != nil {
		return err
	}

	// Create symlink (or copy on Windows) to .pocket/bin/.
	_, err := pocket.CreateSymlink(binary)
	return err
}
