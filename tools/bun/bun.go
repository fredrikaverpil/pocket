// Package bun provides bun runtime integration.
// Bun is a JavaScript runtime used by other tools (e.g., prettier).
package bun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/download"
)

// Name is the binary name for bun.
const Name = "bun"

// Version is the version of bun to install.
// renovate: datasource=github-releases depName=oven-sh/bun extractVersion=^bun-v(?<version>.*)$
const Version = "1.3.6"

// Install ensures bun is available.
var Install = &pk.Task{
	Name:   "install:bun",
	Usage:  "install bun",
	Body:   installBun(),
	Hidden: true,
	Global: true,
}

func installBun() pk.Runnable {
	binDir := pk.FromToolsDir(Name, Version, "bin")
	binaryName := pk.BinaryName(Name)
	binaryPath := filepath.Join(binDir, binaryName)

	url := fmt.Sprintf(
		"https://github.com/oven-sh/bun/releases/download/bun-v%s/bun-%s.zip",
		Version, platformArch(),
	)

	return download.Download(url,
		download.WithDestDir(binDir),
		download.WithFormat("zip"),
		download.WithExtract(download.WithExtractFile(binaryName)),
		download.WithSymlink(),
		download.WithSkipIfExists(binaryPath),
	)
}

func platformArch() string {
	hostOS := pk.HostOS()
	hostArch := pk.HostArch()

	switch hostOS {
	case pk.Darwin:
		if hostArch == pk.ARM64 {
			return "darwin-aarch64"
		}
		return "darwin-x64"
	case pk.Linux:
		if hostArch == pk.ARM64 {
			return "linux-aarch64"
		}
		return "linux-x64"
	case pk.Windows:
		return "windows-x64"
	default:
		return fmt.Sprintf("%s-%s", hostOS, hostArch)
	}
}

// ContentHash returns a short hash for use as a directory name.
// Pass all embedded files to get a content-addressable key.
func ContentHash(data ...[]byte) string {
	h := sha256.New()
	for _, d := range data {
		h.Write(d)
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// IsInstalled reports whether a Node tool is properly installed in the directory.
// It checks that the tool binary exists in node_modules/.bin/.
func IsInstalled(installDir, name string) bool {
	_, err := os.Stat(BinaryPath(installDir, name))
	return err == nil
}

// EnsureInstalled returns a Runnable that skips installFn if the tool is already
// installed. Use this instead of calling IsInstalled manually to ensure the
// check is never forgotten.
func EnsureInstalled(installDir, name string, installFn func(ctx context.Context) error) pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		if IsInstalled(installDir, name) {
			return nil
		}
		return installFn(ctx)
	})
}

// BinaryPath returns the path to a binary installed by bun in the given directory.
func BinaryPath(installDir, binaryName string) string {
	return filepath.Join(installDir, "node_modules", ".bin", pk.BinaryName(binaryName))
}

// InstallFromLockfile installs dependencies from package.json and bun.lock in dir.
// Requires both files for reproducible builds.
func InstallFromLockfile(ctx context.Context, dir string) error {
	packageJSON := filepath.Join(dir, "package.json")
	lockfile := filepath.Join(dir, "bun.lock")

	if _, err := os.Stat(packageJSON); err != nil {
		return fmt.Errorf("package.json not found in %s: %w", dir, err)
	}
	if _, err := os.Stat(lockfile); err != nil {
		return fmt.Errorf("bun.lock not found in %s: %w", dir, err)
	}

	return pk.Exec(ctx, Name, "install", "--cwd", dir, "--frozen-lockfile")
}

// Run executes a package installed via bun.
func Run(ctx context.Context, installDir, packageName string, args ...string) error {
	runArgs := []string{"run", "--cwd", installDir, packageName}
	runArgs = append(runArgs, args...)
	return pk.Exec(ctx, Name, runArgs...)
}
