// Package treesitter provides tree-sitter CLI tool integration.
package treesitter

import (
	"fmt"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/download"
	"github.com/fredrikaverpil/pocket/pk/platform"
)

// Name is the binary name for tree-sitter.
const Name = "tree-sitter"

// Version is the version of tree-sitter to install.
// renovate: datasource=github-releases depName=tree-sitter/tree-sitter
const Version = "0.26.3"

// Install ensures tree-sitter CLI is available.
var Install = pk.NewTask("install:tree-sitter", "install tree-sitter CLI", nil,
	installTreeSitter(),
).Hidden().Global()

func installTreeSitter() pk.Runnable {
	binDir := pk.FromToolsDir("treesitter", Version, "bin")
	binaryName := platform.BinaryName(Name)
	binaryPath := filepath.Join(binDir, binaryName)

	hostOS := platform.HostOS()
	hostArch := platform.HostArch()

	// tree-sitter uses macos instead of darwin
	tsOS := hostOS
	if hostOS == platform.Darwin {
		tsOS = "macos"
	}

	// tree-sitter uses x64 instead of amd64/x86_64
	tsArch := hostArch
	switch hostArch {
	case platform.AMD64:
		tsArch = "x64"
	case platform.ARM64:
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
