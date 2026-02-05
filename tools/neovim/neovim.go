// Package neovim provides Neovim tool integration.
package neovim

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/download"
)

// Name is the binary name for neovim.
const Name = "nvim"

// Version constants for common builds.
const (
	Stable  = "stable"  // Latest stable release
	Nightly = "nightly" // Nightly build
)

// DefaultVersion is the default version to install.
// renovate: datasource=github-releases depName=neovim/neovim
const DefaultVersion = "v0.11.5"

var (
	// InstallStable installs stable Neovim (DefaultVersion).
	InstallStable = Install(Stable)

	// InstallNightly installs nightly Neovim.
	InstallNightly = Install(Nightly)
)

// BinaryPath returns the full path to the neovim binary for a given version.
// Use this instead of Name when running specific versions to avoid symlink collisions.
func BinaryPath(version string) string {
	if version == "" {
		version = DefaultVersion
	}
	resolvedVersion := version
	if version == Stable {
		resolvedVersion = DefaultVersion
	}

	hostOS := pk.HostOS()
	hostArch := pk.HostArch()
	nvimArch := hostArch
	if hostArch == pk.AMD64 {
		nvimArch = pk.X8664
	}

	var plat string
	switch hostOS {
	case pk.Windows:
		if hostArch == pk.ARM64 {
			plat = "win-arm64"
		} else {
			plat = "win64"
		}
	case pk.Darwin:
		plat = fmt.Sprintf("macos-%s", nvimArch)
	default:
		plat = fmt.Sprintf("linux-%s", nvimArch)
	}

	installDir := pk.FromToolsDir("neovim", resolvedVersion)
	binaryName := pk.BinaryName(Name)
	return filepath.Join(installDir, fmt.Sprintf("nvim-%s", plat), "bin", binaryName)
}

// Install creates a task that ensures Neovim is available at the specified version.
// Supported versions: "stable", "nightly", or a specific version like "v0.10.0".
func Install(version string) *pk.Task {
	if version == "" {
		version = DefaultVersion
	}

	taskName := fmt.Sprintf("install:nvim-%s", sanitizeVersion(version))
	return &pk.Task{
		Name:   taskName,
		Usage:  fmt.Sprintf("install neovim %s", version),
		Body:   installNeovim(version),
		Hidden: true,
		Global: true,
	}
}

func sanitizeVersion(version string) string {
	// Replace characters that might be problematic in task names
	return strings.ReplaceAll(version, ".", "-")
}

func installNeovim(version string) pk.Runnable {
	// Resolve "stable" to default version for now
	// (could be enhanced to fetch from GitHub API)
	resolvedVersion := version
	if version == Stable {
		resolvedVersion = DefaultVersion
	}

	// Extract full distribution (includes bin/, lib/, share/ with runtime files)
	// The tarball extracts to nvim-{platform}/ directory
	installDir := pk.FromToolsDir("neovim", resolvedVersion)
	url, format, plat := buildDownloadURL(resolvedVersion)

	binaryName := pk.BinaryName(Name)
	// Binary is at: .pocket/tools/neovim/{version}/nvim-{platform}/bin/nvim
	binaryPath := filepath.Join(installDir, fmt.Sprintf("nvim-%s", plat), "bin", binaryName)

	return pk.Serial(
		download.Download(url,
			download.WithDestDir(installDir),
			download.WithFormat(format),
			// No WithExtract options = extract everything
			download.WithSkipIfExists(binaryPath),
		),
		// Create symlink manually (WithSymlink doesn't work with full extraction)
		createSymlink(binaryPath, version),
	)
}

func createSymlink(binaryPath, version string) pk.Runnable {
	return pk.Do(func(_ context.Context) error {
		binDir := filepath.Dir(binaryPath)

		// On Windows, neovim can't be symlinked because it needs its runtime files
		// relative to the executable. Register the bin directory in PATH instead.
		if pk.HostOS() == pk.Windows {
			pk.RegisterPATH(binDir)
			return nil
		}

		// On Unix, create version-specific symlink to avoid collisions
		// when installing multiple versions in parallel.
		symlinkName := pk.BinaryName(fmt.Sprintf("%s-%s", Name, version))
		_, err := download.CreateSymlinkAs(binaryPath, symlinkName)
		return err
	})
}

func buildDownloadURL(version string) (url, format, plat string) {
	hostOS := pk.HostOS()
	hostArch := pk.HostArch()

	// Neovim uses x86_64 naming (not amd64), but keeps arm64 as-is
	nvimArch := hostArch
	if hostArch == pk.AMD64 {
		nvimArch = pk.X8664
	}

	// Build platform suffix
	switch hostOS {
	case pk.Windows:
		if hostArch == pk.ARM64 {
			plat = "win-arm64"
		} else {
			plat = "win64"
		}
		format = "zip"
	case pk.Darwin:
		plat = fmt.Sprintf("macos-%s", nvimArch)
		format = "tar.gz"
	default: // Linux
		plat = fmt.Sprintf("linux-%s", nvimArch)
		format = "tar.gz"
	}

	// Build extension
	ext := "tar.gz"
	if hostOS == pk.Windows {
		ext = "zip"
	}

	// Build URL
	// nightly uses "nightly" as the tag, stable versions use the version tag
	url = fmt.Sprintf(
		"https://github.com/neovim/neovim/releases/download/%s/nvim-%s.%s",
		version, plat, ext,
	)

	return url, format, plat
}
