package pk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// InstallCargo creates a Runnable that installs a Rust crate using cargo.
// The tool is installed to .pocket/tools/cargo/<name>/<version>/ and
// symlinked to .pocket/bin/ for easy PATH access.
//
// For git-based installs:
//
//	pk.InstallCargo("ts_query_ls",
//	    pk.WithCargoGit("https://github.com/ribru17/ts_query_ls"),
//	)
//
// For crates.io installs:
//
//	pk.InstallCargo("ripgrep", pk.WithCargoVersion("14.1.0"))
func InstallCargo(name string, opts ...CargoOption) Runnable {
	cfg := &cargoConfig{name: name}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// CargoOption configures cargo install behavior.
type CargoOption func(*cargoConfig)

// WithCargoVersion specifies a version for crates.io installs.
func WithCargoVersion(version string) CargoOption {
	return func(c *cargoConfig) {
		c.version = version
	}
}

// WithCargoGit specifies a git repository URL for the crate.
func WithCargoGit(url string) CargoOption {
	return func(c *cargoConfig) {
		c.gitURL = url
	}
}

// WithCargoGitTag specifies a git tag/branch/commit for git-based installs.
func WithCargoGitTag(tag string) CargoOption {
	return func(c *cargoConfig) {
		c.gitTag = tag
	}
}

type cargoConfig struct {
	name    string
	version string
	gitURL  string
	gitTag  string
}

func (c *cargoConfig) run(ctx context.Context) error {
	binaryName := c.name
	if runtime.GOOS == Windows {
		binaryName += ".exe"
	}

	// Determine version string for directory naming.
	versionDir := c.version
	if versionDir == "" && c.gitURL != "" {
		// For git installs without version, use "git" or the tag.
		if c.gitTag != "" {
			versionDir = c.gitTag
		} else {
			versionDir = "latest"
		}
	}
	if versionDir == "" {
		versionDir = "latest"
	}

	// Destination directory: .pocket/tools/cargo/<name>/<version>/
	toolDir := FromToolsDir("cargo", c.name, versionDir)
	toolBinPath := filepath.Join(toolDir, "bin", binaryName)

	// Check if already installed.
	if _, err := os.Stat(toolBinPath); err == nil {
		// Already installed, ensure symlink exists.
		if _, err := CreateSymlink(toolBinPath); err != nil {
			return err
		}
		if Verbose(ctx) {
			Printf(ctx, "  [install] %s already installed\n", binaryName)
		}
		return nil
	}

	// Create tool directory.
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return fmt.Errorf("creating tool directory: %w", err)
	}

	// Build cargo install args.
	args := []string{"install"}

	if c.gitURL != "" {
		args = append(args, "--git", c.gitURL)
		if c.gitTag != "" {
			args = append(args, "--tag", c.gitTag)
		}
	} else if c.version != "" {
		args = append(args, "--version", c.version)
	}

	args = append(args, "--root", toolDir, c.name)

	cmd := exec.CommandContext(ctx, "cargo", args...)
	cmd.Env = os.Environ()
	out := OutputFromContext(ctx)

	if Verbose(ctx) {
		Printf(ctx, "  [install] cargo %s\n", strings.Join(args, " "))
		cmd.Stdout = out.Stdout
		cmd.Stderr = out.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("cargo install %s: %w", c.name, err)
		}
	} else {
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("cargo install %s: %w\n%s", c.name, err, output)
		}
	}

	// Create symlink.
	if _, err := CreateSymlink(toolBinPath); err != nil {
		return err
	}

	if Verbose(ctx) {
		Printf(ctx, "  [install] linked %s\n", toolBinPath)
	}

	return nil
}
