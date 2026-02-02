// Package golang provides Go package installation integration.
//
// # Usage
//
// Install Go packages using `go install`:
//
//	golang.Install("github.com/golangci/golangci-lint/v2/cmd/golangci-lint", "v2.1.6")
//
// Tools are installed to .pocket/tools/go/<pkg>/<version>/ and
// symlinked to .pocket/bin/ for easy PATH access.
package golang

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/download"
	"github.com/fredrikaverpil/pocket/pk/pcontext"
	"github.com/fredrikaverpil/pocket/pk/platform"
)

// Install creates a Runnable that installs a Go package using `go install`.
// The tool is installed to .pocket/tools/go/<pkg>/<version>/ and
// symlinked to .pocket/bin/ for easy PATH access.
//
// The pkg should be the full module path, e.g.:
//
//	golang.Install("github.com/golangci/golangci-lint/v2/cmd/golangci-lint", "v2.1.6")
//
// The binary name is extracted from the last path segment of pkg.
func Install(pkg, version string) pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return install(ctx, pkg, version)
	})
}

func install(ctx context.Context, pkg, version string) error {
	// Extract binary name from package path.
	binaryName := binaryName(pkg)
	if runtime.GOOS == platform.Windows {
		binaryName += ".exe"
	}

	// Destination directory: .pocket/tools/go/<pkg>/<version>/
	toolDir := pk.FromToolsDir("go", pkg, version)
	toolBinPath := filepath.Join(toolDir, binaryName)

	// Check if already installed.
	if _, err := os.Stat(toolBinPath); err == nil {
		// Already installed, ensure symlink exists.
		if _, err := download.CreateSymlink(toolBinPath); err != nil {
			return err
		}
		if pcontext.Verbose(ctx) {
			pk.Printf(ctx, "  [install] %s@%s already installed\n", binaryName, version)
		}
		return nil
	}

	// Create tool directory.
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return fmt.Errorf("creating tool directory: %w", err)
	}

	// Run go install with GOBIN set.
	pkgWithVersion := pkg + "@" + version
	cmd := exec.CommandContext(ctx, "go", "install", pkgWithVersion)
	cmd.Env = append(os.Environ(), "GOBIN="+toolDir)

	if pcontext.Verbose(ctx) {
		pk.Printf(ctx, "  [install] go install %s\n", pkgWithVersion)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go install %s: %w", pkgWithVersion, err)
		}
	} else {
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("go install %s: %w\n%s", pkgWithVersion, err, output)
		}
	}

	// Create symlink (or copy on Windows).
	if _, err := download.CreateSymlink(toolBinPath); err != nil {
		return err
	}

	if pcontext.Verbose(ctx) {
		pk.Printf(ctx, "  [install] linked %s\n", toolBinPath)
	}

	return nil
}

// binaryName extracts the binary name from a Go package path.
func binaryName(pkg string) string {
	parts := strings.Split(pkg, "/")
	// If path ends with /cmd/<name>, use <name>
	if len(parts) >= 2 && parts[len(parts)-2] == "cmd" {
		return parts[len(parts)-1]
	}
	// Otherwise use last non-version component.
	for i := len(parts) - 1; i >= 0; i-- {
		if !strings.HasPrefix(parts[i], "v") || !isVersion(parts[i]) {
			return parts[i]
		}
	}
	return parts[len(parts)-1]
}

func isVersion(s string) bool {
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
