package tool

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fredrikaverpil/bld"
)

// GoInstall installs a Go binary using 'go install'.
// The binary is installed to .bld/tools/go/<pkg>/<version>/<binary-name>
// and symlinked to .bld/bin/<binary-name>.
// Returns the path to the installed binary.
func GoInstall(ctx context.Context, pkg, version string) (string, error) {
	// Determine binary name from package path
	binaryName := goBinaryName(pkg)

	// Destination directory: .bld/tools/go/<pkg>/<version>/
	toolDir := bld.FromToolsDir("go", pkg, version)
	binaryPath := filepath.Join(toolDir, binaryName)

	// Check if already installed
	if _, err := os.Stat(binaryPath); err == nil {
		// Already installed, ensure symlink exists
		if _, err := CreateSymlink(binaryPath); err != nil {
			return "", err
		}
		return binaryPath, nil
	}

	// Create tool directory
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return "", fmt.Errorf("create tool dir: %w", err)
	}

	// Run go install with GOBIN set
	pkgWithVersion := pkg + "@" + version
	cmd := exec.CommandContext(ctx, "go", "install", pkgWithVersion)
	cmd.Env = append(os.Environ(), "GOBIN="+toolDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go install %s: %w", pkgWithVersion, err)
	}

	// Create symlink
	if _, err := CreateSymlink(binaryPath); err != nil {
		return "", err
	}

	return binaryPath, nil
}

// GoInstallWithModfile installs a Go binary using the version from a go.mod file.
func GoInstallWithModfile(ctx context.Context, pkg, modfile string) (string, error) {
	version, err := getVersionFromModfile(modfile, pkg)
	if err != nil {
		return "", err
	}
	return GoInstall(ctx, pkg, version)
}

// goBinaryName extracts the binary name from a Go package path.
// E.g., "github.com/golangci/golangci-lint/v2/cmd/golangci-lint" -> "golangci-lint".
// E.g., "github.com/google/go-licenses/v2" -> "go-licenses".
func goBinaryName(pkg string) string {
	// If path ends with /cmd/<name>, use <name>
	parts := strings.Split(pkg, "/")
	if len(parts) >= 2 && parts[len(parts)-2] == "cmd" {
		return parts[len(parts)-1]
	}

	// Otherwise use last non-version component
	for i := len(parts) - 1; i >= 0; i-- {
		if !strings.HasPrefix(parts[i], "v") || !isVersion(parts[i]) {
			return parts[i]
		}
	}
	return parts[len(parts)-1]
}

func isVersion(s string) bool {
	if len(s) < 2 {
		return false
	}
	if s[0] != 'v' {
		return false
	}
	for _, c := range s[1:] {
		if c != '.' && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func getVersionFromModfile(modfile, pkg string) (string, error) {
	data, err := os.ReadFile(modfile)
	if err != nil {
		return "", fmt.Errorf("read modfile: %w", err)
	}

	// Simple parsing - look for the package in require blocks
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, pkg) || strings.Contains(line, pkg+" ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return fields[1], nil
			}
		}
	}

	return "", fmt.Errorf("package %s not found in %s", pkg, modfile)
}
