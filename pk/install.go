package pk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// FromPocketDir constructs an absolute path within the .pocket directory.
//
//	FromPocketDir()           → "/path/to/repo/.pocket"
//	FromPocketDir("bin")      → "/path/to/repo/.pocket/bin"
//	FromPocketDir("tools")    → "/path/to/repo/.pocket/tools"
func FromPocketDir(elem ...string) string {
	parts := append([]string{findGitRoot(), ".pocket"}, elem...)
	return filepath.Join(parts...)
}

// FromToolsDir constructs an absolute path within .pocket/tools.
//
//	FromToolsDir()                    → "/path/to/repo/.pocket/tools"
//	FromToolsDir("go", "pkg", "v1.0") → "/path/to/repo/.pocket/tools/go/pkg/v1.0"
func FromToolsDir(elem ...string) string {
	parts := append([]string{"tools"}, elem...)
	return FromPocketDir(parts...)
}

// FromBinDir constructs an absolute path within .pocket/bin.
//
//	FromBinDir()             → "/path/to/repo/.pocket/bin"
//	FromBinDir("golangci-lint") → "/path/to/repo/.pocket/bin/golangci-lint"
func FromBinDir(elem ...string) string {
	parts := append([]string{"bin"}, elem...)
	return FromPocketDir(parts...)
}

// InstallGo creates a Runnable that installs a Go package.
// The tool is installed to .pocket/tools/go/<pkg>/<version>/ and
// symlinked to .pocket/bin/ for easy PATH access.
//
// The pkg should be the full module path, e.g.:
//
//	pk.InstallGo("github.com/golangci/golangci-lint/v2/cmd/golangci-lint", "v2.1.6")
//
// The binary name is extracted from the last path segment of pkg.
func InstallGo(pkg, version string) Runnable {
	return &goInstaller{pkg: pkg, version: version}
}

type goInstaller struct {
	pkg     string
	version string
}

func (g *goInstaller) run(ctx context.Context) error {
	// Extract binary name from package path (last segment)
	binaryName := filepath.Base(g.pkg)

	// Check if already installed at this version
	binPath := FromBinDir(binaryName)
	toolBinDir := FromToolsDir("go", g.pkg, g.version, "bin")
	toolBinPath := filepath.Join(toolBinDir, binaryName)

	// If symlink exists and points to correct version, skip
	if target, err := os.Readlink(binPath); err == nil {
		if target == toolBinPath {
			if Verbose(ctx) {
				Printf(ctx, "  [install] %s@%s already installed\n", binaryName, g.version)
			}
			return nil
		}
	}

	// Create tool directory
	if err := os.MkdirAll(toolBinDir, 0o755); err != nil {
		return fmt.Errorf("creating tool directory: %w", err)
	}

	// Run go install with GOBIN set
	pkgWithVersion := g.pkg + "@" + g.version
	cmd := exec.CommandContext(ctx, "go", "install", pkgWithVersion)
	cmd.Env = append(os.Environ(), "GOBIN="+toolBinDir)
	out := OutputFromContext(ctx)

	if Verbose(ctx) {
		Printf(ctx, "  [install] go install %s\n", pkgWithVersion)
		cmd.Stdout = out.Stdout
		cmd.Stderr = out.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go install %s: %w", pkgWithVersion, err)
		}
	} else {
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("go install %s: %w\n%s", pkgWithVersion, err, output)
		}
	}

	// Create bin directory if needed
	binDir := FromBinDir()
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("creating bin directory: %w", err)
	}

	// Remove existing symlink if present
	_ = os.Remove(binPath)

	// Create symlink from .pocket/bin/<name> to .pocket/tools/go/<pkg>/<version>/bin/<name>
	if err := os.Symlink(toolBinPath, binPath); err != nil {
		return fmt.Errorf("creating symlink: %w", err)
	}

	if Verbose(ctx) {
		Printf(ctx, "  [install] linked %s -> %s\n", binPath, toolBinPath)
	}

	return nil
}
