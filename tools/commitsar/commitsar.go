// Package commitsar provides commitsar tool integration.
// Commitsar checks that commits comply with conventional commit standards.
package commitsar

import (
	"fmt"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/download"
)

// Name is the binary name for commitsar.
const Name = "commitsar"

// Version is the version of commitsar to install.
// renovate: datasource=github-releases depName=aevea/commitsar
const Version = "1.0.2"

// Install ensures commitsar is available.
var Install = &pk.Task{
	Name:   "install:commitsar",
	Usage:  "install commitsar",
	Body:   installCommitsar(),
	Hidden: true,
	Global: true,
}

func installCommitsar() pk.Runnable {
	binDir := pk.FromToolsDir(Name, Version, "bin")
	binaryName := pk.BinaryName(Name)
	binaryPath := filepath.Join(binDir, binaryName)

	url := fmt.Sprintf(
		"https://github.com/aevea/commitsar/releases/download/v%s/commitsar_%s_%s_%s.tar.gz",
		Version, Version, pk.HostOS(), pk.HostArch(),
	)

	return download.Download(url,
		download.WithDestDir(binDir),
		download.WithFormat("tar.gz"),
		download.WithExtract(download.WithExtractFile(binaryName)),
		download.WithSymlink(),
		download.WithSkipIfExists(binaryPath),
	)
}
