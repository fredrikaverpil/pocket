// Package bun provides bun runtime integration.
// Bun is a JavaScript runtime used by other tools (e.g., prettier).
package bun

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/fredrikaverpil/pocket"
)

// Name is the binary name for bun.
const Name = "bun"

// renovate: datasource=github-releases depName=oven-sh/bun extractVersion=^bun-v(?<version>.*)$
const Version = "1.3.6"

// Install ensures bun is available.
var Install = pocket.Func("install:bun", "ensure bun is available", install).Hidden()

func install(ctx context.Context) error {
	binDir := pocket.FromToolsDir(Name, Version, "bin")
	binaryName := pocket.BinaryName(Name)
	binaryPath := filepath.Join(binDir, binaryName)

	url := fmt.Sprintf(
		"https://github.com/oven-sh/bun/releases/download/bun-v%s/bun-%s.zip",
		Version, platformArch(),
	)

	pocket.Printf(ctx, "Installing bun %s...\n", Version)
	return pocket.Download(ctx, url,
		pocket.WithDestDir(binDir),
		pocket.WithFormat("zip"),
		pocket.WithExtract(pocket.WithExtractFile(binaryName)),
		pocket.WithSymlink(),
		pocket.WithSkipIfExists(binaryPath),
	)
}

func platformArch() string {
	hostOS := pocket.HostOS()
	hostArch := pocket.HostArch()

	switch hostOS {
	case pocket.Darwin:
		if hostArch == pocket.ARM64 {
			return "darwin-aarch64"
		}
		return "darwin-x64"
	case pocket.Linux:
		if hostArch == pocket.ARM64 {
			return "linux-aarch64"
		}
		return "linux-x64"
	case pocket.Windows:
		return "windows-x64"
	default:
		return fmt.Sprintf("%s-%s", hostOS, hostArch)
	}
}

// BinaryPath returns the path to a binary installed by bun in the given directory.
// On Windows, it appends .exe to the binary name.
func BinaryPath(installDir, binaryName string) string {
	return filepath.Join(installDir, "node_modules", ".bin", pocket.BinaryName(binaryName))
}
