// Package nvim provides Neovim tool integration.
// Used for running plenary tests and other Neovim-based operations.
package nvim

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/fredrikaverpil/pocket"
)

// Name is the binary name for nvim.
const Name = "nvim"

// renovate: datasource=github-releases depName=neovim/neovim
const Version = "0.11.5"

// Install ensures nvim is available.
var Install = pocket.Task("install:nvim", "install neovim",
	installNvim(),
	pocket.AsHidden(),
)

func installNvim() pocket.Runnable {
	binDir := pocket.FromToolsDir("nvim", Version)
	binaryName := pocket.BinaryName("nvim")

	platform := platformArch()
	ext := "tar.gz"
	if runtime.GOOS == pocket.Windows {
		ext = "zip"
	}

	url := fmt.Sprintf(
		"https://github.com/neovim/neovim/releases/download/v%s/nvim-%s.%s",
		Version, platform, ext,
	)

	// Neovim archives extract to nvim-<platform>/bin/nvim
	archiveDir := fmt.Sprintf("nvim-%s", platform)
	binaryInArchive := filepath.Join(archiveDir, "bin", binaryName)
	extractedBinaryPath := filepath.Join(binDir, archiveDir, "bin", binaryName)

	return pocket.Serial(
		pocket.Download(url,
			pocket.WithDestDir(binDir),
			pocket.WithFormat(ext),
			pocket.WithExtract(pocket.WithExtractFile(binaryInArchive)),
			pocket.WithSkipIfExists(extractedBinaryPath),
		),
		// Create symlink to the binary inside the extracted directory.
		pocket.Do(func(_ context.Context) error {
			_, err := pocket.CreateSymlink(extractedBinaryPath)
			return err
		}),
	)
}

func platformArch() string {
	switch runtime.GOOS {
	case pocket.Darwin:
		if runtime.GOARCH == pocket.ARM64 {
			return "macos-arm64"
		}
		return "macos-x86_64"
	case pocket.Linux:
		if runtime.GOARCH == pocket.ARM64 {
			return "linux-arm64"
		}
		return "linux-x86_64"
	case pocket.Windows:
		if runtime.GOARCH == pocket.ARM64 {
			return "win-arm64"
		}
		return "win64"
	default:
		return fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	}
}
