// Package pagefind provides pagefind tool integration.
// Pagefind is a static search library for websites.
package pagefind

import (
	"fmt"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/download"
	"github.com/fredrikaverpil/pocket/pk/platform"
	"github.com/fredrikaverpil/pocket/pk/repopath"
)

// Name is the binary name for pagefind.
const Name = "pagefind"

// Version is the version of pagefind to install.
// renovate: datasource=github-releases depName=CloudCannon/pagefind
const Version = "1.4.0"

// Install ensures pagefind (extended) is available.
var Install = &pk.Task{
	Name:   "install:pagefind",
	Usage:  "install pagefind",
	Body:   installPagefind(),
	Hidden: true,
	Global: true,
}

func installPagefind() pk.Runnable {
	binDir := repopath.FromToolsDir(Name, Version, "bin")
	binaryName := platform.BinaryName(Name)
	binaryPath := filepath.Join(binDir, binaryName)

	url := fmt.Sprintf(
		"https://github.com/CloudCannon/pagefind/releases/download/v%s/pagefind_extended-v%s-%s.tar.gz",
		Version, Version, platformTarget(),
	)

	return download.Download(url,
		download.WithDestDir(binDir),
		download.WithFormat("tar.gz"),
		download.WithExtract(download.WithExtractFile(binaryName)),
		download.WithSymlink(),
		download.WithSkipIfExists(binaryPath),
	)
}

func platformTarget() string {
	hostOS := platform.HostOS()
	hostArch := platform.HostArch()

	archName := platform.ArchToX8664(hostArch)

	switch hostOS {
	case platform.Darwin:
		return archName + "-apple-darwin"
	case platform.Linux:
		return archName + "-unknown-linux-musl"
	case platform.Windows:
		return archName + "-pc-windows-msvc"
	default:
		return fmt.Sprintf("%s-%s", archName, hostOS)
	}
}
