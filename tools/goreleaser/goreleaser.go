// Package goreleaser provides goreleaser tool integration.
package goreleaser

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/download"
)

//go:embed goreleaser.yml
var defaultConfig []byte

// Name is the binary name for goreleaser.
const Name = "goreleaser"

// Version is the version of goreleaser to install.
// renovate: datasource=github-releases depName=goreleaser/goreleaser
const Version = "2.14.0"

// Install ensures goreleaser is available.
var Install = &pk.Task{
	Name:   "install:goreleaser",
	Usage:  "install goreleaser",
	Body:   installGoreleaser(),
	Hidden: true,
	Global: true,
}

func installGoreleaser() pk.Runnable {
	binDir := pk.FromToolsDir(Name, Version, "bin")
	binaryName := pk.BinaryName(Name)
	binaryPath := filepath.Join(binDir, binaryName)

	hostOS := pk.HostOS()
	hostArch := pk.HostArch()

	// GoReleaser uses title-case OS names and x86_64 for amd64.
	osName := pk.OSToTitle(hostOS)
	archName := hostArch
	if hostArch == pk.AMD64 {
		archName = pk.X8664
	}

	url := fmt.Sprintf(
		"https://github.com/goreleaser/goreleaser/releases/download/v%s/goreleaser_%s_%s.%s",
		Version, osName, archName, pk.DefaultArchiveFormat(),
	)

	return download.Download(url,
		download.WithDestDir(binDir),
		download.WithFormat(pk.DefaultArchiveFormat()),
		download.WithExtract(download.WithExtractFile(binaryName)),
		download.WithSymlink(),
		download.WithSkipIfExists(binaryPath),
	)
}

// WriteDefaultConfig writes a default .goreleaser.yml to the git root.
// Only writes if the file doesn't already exist. Returns the path written to.
func WriteDefaultConfig() (string, error) {
	destPath := pk.FromGitRoot(".goreleaser.yml")
	if _, err := os.Stat(destPath); err == nil {
		return destPath, nil
	}
	if err := os.WriteFile(destPath, defaultConfig, 0o644); err != nil {
		return "", fmt.Errorf("write .goreleaser.yml: %w", err)
	}
	return destPath, nil
}
