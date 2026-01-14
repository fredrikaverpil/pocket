package pocket

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// GoInstall installs a Go binary using 'go install'.
// The binary is installed to .pocket/tools/go/<pkg>/<version>/
// and symlinked to .pocket/bin/.
//
// Example:
//
//	func install(ctx context.Context, tc *pocket.TaskContext) error {
//	    tc.Out.Printf("Installing govulncheck %s...\n", version)
//	    _, err := pocket.GoInstall(ctx, tc, "golang.org/x/vuln/cmd/govulncheck", version)
//	    return err
//	}
func GoInstall(ctx context.Context, tc *TaskContext, pkg, version string) (string, error) {
	// Determine binary name from package path.
	binaryName := goBinaryName(pkg)
	if runtime.GOOS == Windows {
		binaryName += ".exe"
	}

	// Destination directory: .pocket/tools/go/<pkg>/<version>/
	toolDir := FromToolsDir("go", pkg, version)
	binaryPath := filepath.Join(toolDir, binaryName)

	// Check if already installed.
	if _, err := os.Stat(binaryPath); err == nil {
		// Already installed, ensure symlink exists.
		if _, err := CreateSymlink(binaryPath); err != nil {
			return "", err
		}
		return binaryPath, nil
	}

	// Create tool directory.
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return "", fmt.Errorf("create tool dir: %w", err)
	}

	// Run go install with GOBIN set.
	pkgWithVersion := pkg + "@" + version
	tc.Out.Printf("  go install %s\n", pkgWithVersion)

	cmd := tc.Command(ctx, "go", "install", pkgWithVersion)
	cmd.Env = append(cmd.Environ(), "GOBIN="+toolDir)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go install %s: %w", pkgWithVersion, err)
	}

	// Create symlink.
	if _, err := CreateSymlink(binaryPath); err != nil {
		return "", err
	}

	return binaryPath, nil
}

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

// VenvBinaryPath returns the cross-platform path to a binary in a Python venv.
func VenvBinaryPath(venvDir, name string) string {
	if runtime.GOOS == Windows {
		return filepath.Join(venvDir, "Scripts", name+".exe")
	}
	return filepath.Join(venvDir, "bin", name)
}

// goBinaryName extracts the binary name from a Go package path.
func goBinaryName(pkg string) string {
	parts := strings.Split(pkg, "/")
	// If path ends with /cmd/<name>, use <name>
	if len(parts) >= 2 && parts[len(parts)-2] == "cmd" {
		return parts[len(parts)-1]
	}
	// Otherwise use last non-version component
	for i := len(parts) - 1; i >= 0; i-- {
		if !strings.HasPrefix(parts[i], "v") || !isGoVersion(parts[i]) {
			return parts[i]
		}
	}
	return parts[len(parts)-1]
}

func isGoVersion(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	for _, c := range s[1:] {
		if c != '.' && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// ensureToolsGoMod creates .pocket/tools/go.mod if it doesn't exist.
// This prevents go mod tidy from scanning downloaded tools which may
// contain test files with relative imports that break module mode.
func ensureToolsGoMod() error {
	toolsDir := FromToolsDir()
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		return fmt.Errorf("create tools dir: %w", err)
	}

	goModPath := filepath.Join(toolsDir, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		return nil // Already exists
	}

	// Read Go version from .pocket/go.mod
	goVersion, err := GoVersionFromDir(FromPocketDir())
	if err != nil {
		return err
	}

	content := fmt.Sprintf(`// This file prevents go mod tidy from scanning downloaded tools.
// Downloaded tools (like Go SDK) contain test files with relative imports
// that break module mode.

module pocket-tools

go %s
`, goVersion)

	if err := os.WriteFile(goModPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write tools/go.mod: %w", err)
	}
	return nil
}
