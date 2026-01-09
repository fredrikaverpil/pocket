package tool

import (
	"path/filepath"
	"runtime"
)

// VenvBinaryPath returns the path to a binary in a Python virtual environment.
// On Windows, binaries are in Scripts/ with .exe extension.
// On Unix, binaries are in bin/ without extension.
func VenvBinaryPath(venvDir, name string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", name+".exe")
	}
	return filepath.Join(venvDir, "bin", name)
}
