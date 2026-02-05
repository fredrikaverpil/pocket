// Package treesitter provides tree-sitter CLI tool integration.
package treesitter

import (
	"fmt"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/download"
)

// Name is the binary name for tree-sitter.
const Name = "tree-sitter"

// Version is the version of tree-sitter to install.
// renovate: datasource=github-releases depName=tree-sitter/tree-sitter
const Version = "0.26.3"

// Install ensures tree-sitter CLI is available.
var Install = pk.NewTask(pk.TaskConfig{
	Name:   "install:tree-sitter",
	Usage:  "install tree-sitter CLI",
	Body:   installTreeSitter(),
	Hidden: true,
	Global: true,
})

func installTreeSitter() pk.Runnable {
	binDir := pk.FromToolsDir("treesitter", Version, "bin")
	binaryName := pk.BinaryName(Name)
	binaryPath := filepath.Join(binDir, binaryName)

	hostOS := pk.HostOS()
	hostArch := pk.HostArch()

	// tree-sitter uses macos instead of darwin
	tsOS := hostOS
	if hostOS == pk.Darwin {
		tsOS = "macos"
	}

	// tree-sitter uses x64 instead of amd64/x86_64
	tsArch := hostArch
	switch hostArch {
	case pk.AMD64:
		tsArch = "x64"
	case pk.ARM64:
		tsArch = "arm64"
	}

	// Asset name: tree-sitter-{os}-{arch}.gz
	assetName := fmt.Sprintf("tree-sitter-%s-%s.gz", tsOS, tsArch)
	url := fmt.Sprintf(
		"https://github.com/tree-sitter/tree-sitter/releases/download/v%s/%s",
		Version, assetName,
	)

	return download.Download(url,
		download.WithDestDir(binDir),
		download.WithFormat("gz"),
		download.WithOutputName(binaryName),
		download.WithSymlink(),
		download.WithSkipIfExists(binaryPath),
	)
}
