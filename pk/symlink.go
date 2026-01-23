package pk

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// CreateSymlink creates a symlink in .pocket/bin/ pointing to the given binary.
// On Windows, it copies the file instead since symlinks require admin privileges.
// Returns the path to the symlink (or copy on Windows).
func CreateSymlink(binaryPath string) (string, error) {
	binDir := FromBinDir()
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", fmt.Errorf("create bin dir: %w", err)
	}

	// Ensure tools/go.mod exists to prevent go mod tidy issues.
	if err := ensureToolsGoMod(); err != nil {
		return "", err
	}

	name := filepath.Base(binaryPath)
	linkPath := filepath.Join(binDir, name)

	// Remove existing file/symlink if it exists.
	if _, err := os.Lstat(linkPath); err == nil {
		if err := os.Remove(linkPath); err != nil {
			return "", fmt.Errorf("remove existing file: %w", err)
		}
	}

	// On Windows, copy the file instead of creating a symlink.
	if runtime.GOOS == Windows {
		if err := CopyFile(binaryPath, linkPath); err != nil {
			return "", fmt.Errorf("copy binary: %w", err)
		}
		return linkPath, nil
	}

	// Create relative symlink on Unix.
	relPath, err := filepath.Rel(binDir, binaryPath)
	if err != nil {
		return "", fmt.Errorf("compute relative path: %w", err)
	}

	if err := os.Symlink(relPath, linkPath); err != nil {
		return "", fmt.Errorf("create symlink: %w", err)
	}

	return linkPath, nil
}

// CreateSymlinkWithCompanions creates a symlink in .pocket/bin/ and copies companion files.
// On Windows, this copies both the binary and any companion files (like DLLs) from the same directory.
// On Unix, it creates a symlink for the binary only (companion files are typically not needed).
// The companions parameter specifies glob patterns for files to copy (e.g., "*.dll").
func CreateSymlinkWithCompanions(binaryPath string, companions ...string) (string, error) {
	linkPath, err := CreateSymlink(binaryPath)
	if err != nil {
		return "", err
	}

	// On Windows, copy companion files (DLLs, etc.) to the bin directory.
	if runtime.GOOS == Windows && len(companions) > 0 {
		srcDir := filepath.Dir(binaryPath)
		binDir := FromBinDir()

		for _, pattern := range companions {
			matches, err := filepath.Glob(filepath.Join(srcDir, pattern))
			if err != nil {
				return "", fmt.Errorf("glob companion files: %w", err)
			}
			for _, match := range matches {
				name := filepath.Base(match)
				dst := filepath.Join(binDir, name)
				// Skip if it's the main binary (already copied).
				if name == filepath.Base(binaryPath) {
					continue
				}
				if err := CopyFile(match, dst); err != nil {
					return "", fmt.Errorf("copy companion %s: %w", name, err)
				}
			}
		}
	}

	return linkPath, nil
}

// CopyFile copies a file from src to dst.
func CopyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		return fmt.Errorf("write destination: %w", err)
	}
	return nil
}

// ensureToolsGoMod creates .pocket/tools/go.mod if it doesn't exist.
// This prevents go mod tidy from scanning downloaded tools.
func ensureToolsGoMod() error {
	toolsDir := FromToolsDir()
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		return fmt.Errorf("create tools dir: %w", err)
	}

	goModPath := filepath.Join(toolsDir, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		return nil // Already exists.
	}

	// Read Go version from .pocket/go.mod.
	goVersion, err := goVersionFromDir(FromPocketDir())
	if err != nil {
		// Fallback to a reasonable default if we can't read go.mod.
		goVersion = "1.23"
	}

	content := fmt.Sprintf(`// This file prevents go mod tidy from scanning downloaded tools.

module pocket-tools

go %s
`, goVersion)

	if err := os.WriteFile(goModPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write tools/go.mod: %w", err)
	}
	return nil
}

// goVersionFromDir reads the Go version from go.mod in the given directory.
func goVersionFromDir(dir string) (string, error) {
	goModPath := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	// Simple parsing: find "go X.Y" line.
	for _, line := range filepath.SplitList(string(data)) {
		if len(line) > 3 && line[:3] == "go " {
			return line[3:], nil
		}
	}

	return "", fmt.Errorf("go version not found in %s", goModPath)
}
