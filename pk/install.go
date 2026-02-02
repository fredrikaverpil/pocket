package pk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fredrikaverpil/pocket/pk/pcontext"
	"github.com/fredrikaverpil/pocket/pk/platform"
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
	// Extract binary name from package path.
	binaryName := goBinaryName(g.pkg)
	if runtime.GOOS == platform.Windows {
		binaryName += ".exe"
	}

	// Destination directory: .pocket/tools/go/<pkg>/<version>/
	toolDir := FromToolsDir("go", g.pkg, g.version)
	toolBinPath := filepath.Join(toolDir, binaryName)

	// Check if already installed.
	if _, err := os.Stat(toolBinPath); err == nil {
		// Already installed, ensure symlink exists.
		if _, err := CreateSymlink(toolBinPath); err != nil {
			return err
		}
		if pcontext.Verbose(ctx) {
			Printf(ctx, "  [install] %s@%s already installed\n", binaryName, g.version)
		}
		return nil
	}

	// Create tool directory.
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return fmt.Errorf("creating tool directory: %w", err)
	}

	// Run go install with GOBIN set.
	pkgWithVersion := g.pkg + "@" + g.version
	cmd := exec.CommandContext(ctx, "go", "install", pkgWithVersion)
	cmd.Env = append(os.Environ(), "GOBIN="+toolDir)
	out := outputFromContext(ctx)

	if pcontext.Verbose(ctx) {
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

	// Create symlink (or copy on Windows).
	if _, err := CreateSymlink(toolBinPath); err != nil {
		return err
	}

	if pcontext.Verbose(ctx) {
		Printf(ctx, "  [install] linked %s\n", toolBinPath)
	}

	return nil
}

// goBinaryName extracts the binary name from a Go package path.
func goBinaryName(pkg string) string {
	parts := strings.Split(pkg, "/")
	// If path ends with /cmd/<name>, use <name>
	if len(parts) >= 2 && parts[len(parts)-2] == "cmd" {
		return parts[len(parts)-1]
	}
	// Otherwise use last non-version component.
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
