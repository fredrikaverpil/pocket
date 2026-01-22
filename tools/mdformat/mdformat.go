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
var Install = pk.NewTask("install:mdformat", "install mdformat", nil,
	pk.Serial(uv.Install, installMdformat()),
).Hidden()

func installMdformat() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		venvDir := pk.FromToolsDir("mdformat", Version())
		binary := uv.BinaryPath(venvDir, "mdformat")

		// Skip if already installed.
		if _, err := os.Stat(binary); err == nil {
			_, err := pk.CreateSymlink(binary)
			return err
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
		if err := uv.PipInstallRequirements(ctx, venvDir, reqPath); err != nil {
			return err
		}

		// Create symlink to .pocket/bin/.
		_, err := pk.CreateSymlink(binary)
		return err
	})
}
