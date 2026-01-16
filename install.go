package pocket

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// InstallGo creates a Runnable that installs a Go binary using 'go install'.
// The binary is installed to .pocket/tools/go/<pkg>/<version>/
// and symlinked to .pocket/bin/.
//
// Example:
//
//	var Install = pocket.Task("install:linter", "install linter",
//	    pocket.InstallGo("github.com/golangci/golangci-lint/cmd/golangci-lint", version),
//	).Hidden()
func InstallGo(pkg, version string) Runnable {
	return Do(func(ctx context.Context) error {
		return installGo(ctx, pkg, version)
	})
}

// installGo is the internal implementation of InstallGo.
func installGo(ctx context.Context, pkg, version string) error {
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
			return err
		}
		return nil
	}

	// Create tool directory.
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return fmt.Errorf("create tool dir: %w", err)
	}

	// Run go install with GOBIN set.
	pkgWithVersion := pkg + "@" + version
	Printf(ctx, "  go install %s\n", pkgWithVersion)

	ec := getExecContext(ctx)
	cmd := newCommand(ctx, "go", "install", pkgWithVersion)
	cmd.Stdout = ec.out.Stdout
	cmd.Stderr = ec.out.Stderr
	cmd.Env = append(cmd.Environ(), "GOBIN="+toolDir)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go install %s: %w", pkgWithVersion, err)
	}

	// Create symlink.
	if _, err := CreateSymlink(binaryPath); err != nil {
		return err
	}

	return nil
}

// ToolConfig describes how to find or create a tool's configuration file.
type ToolConfig struct {
	// UserFiles are filenames to search for in the repo root.
	// Checked in order; first match wins.
	UserFiles []string

	// DefaultFile is the filename for the bundled default config,
	// written to .pocket/tools/<name>/ if no user config exists.
	DefaultFile string

	// DefaultData is the bundled default configuration content.
	DefaultData []byte
}

// ConfigPath returns the path to a tool's config file.
//
// For each path in UserFiles:
//   - Absolute paths are checked as-is (use FromGitRoot() for repo-root configs)
//   - Relative paths are checked in the task's current directory (from Path(ctx))
//
// If no user config is found, writes DefaultData to .pocket/tools/<name>/<DefaultFile>.
// Returns empty string and no error if cfg is empty.
//
// Example:
//
//	var golangciConfig = pocket.ToolConfig{
//	    UserFiles:   []string{".golangci.yml", ".golangci.yaml"},
//	    DefaultFile: "golangci.yml",
//	    DefaultData: defaultConfig,
//	}
//
//	func lint(ctx context.Context) error {
//	    configPath, err := pocket.ConfigPath(ctx, "golangci-lint", golangciConfig)
//	    if err != nil {
//	        return err
//	    }
//	    return pocket.Exec(ctx, "golangci-lint", "run", "-c", configPath)
//	}
func ConfigPath(ctx context.Context, toolName string, cfg ToolConfig) (string, error) {
	if len(cfg.UserFiles) == 0 && cfg.DefaultFile == "" {
		return "", nil
	}

	// Check for user config files.
	// Absolute paths are used as-is, relative paths are resolved from task CWD.
	cwd := FromGitRoot(Path(ctx))
	for _, configName := range cfg.UserFiles {
		var configPath string
		if filepath.IsAbs(configName) {
			configPath = configName
		} else {
			configPath = filepath.Join(cwd, configName)
		}
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	// No default config provided, return empty.
	if cfg.DefaultFile == "" || len(cfg.DefaultData) == 0 {
		return "", nil
	}

	// Write bundled config to .pocket/tools/<name>/<default-file>.
	configDir := FromToolsDir(toolName)
	configPath := filepath.Join(configDir, cfg.DefaultFile)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			return "", fmt.Errorf("create config dir: %w", err)
		}
		if err := os.WriteFile(configPath, cfg.DefaultData, 0o644); err != nil {
			return "", fmt.Errorf("write default config: %w", err)
		}
	}

	return configPath, nil
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
