// Package commitsar provides commitsar tool integration.
// Commitsar checks that commits comply with conventional commit standards.
package commitsar

import (
	"fmt"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/download"
	"github.com/fredrikaverpil/pocket/pk/platform"
)

// Name is the binary name for commitsar.
const Name = "commitsar"

// Version is the version of commitsar to install.
// renovate: datasource=github-releases depName=aevea/commitsar
const Version = "1.0.2"

// Install ensures commitsar is available.
var Install = pk.NewTask("install:commitsar", "install commitsar", nil,
	installCommitsar(),
).Hidden().Global()

func installCommitsar() pk.Runnable {
	binDir := pk.FromToolsDir(Name, Version, "bin")
	binaryName := platform.BinaryName(Name)
	binaryPath := filepath.Join(binDir, binaryName)

	url := fmt.Sprintf(
		"https://github.com/aevea/commitsar/releases/download/v%s/commitsar_%s_%s_%s.tar.gz",
		Version, Version, platform.HostOS(), platform.HostArch(),
	)

	return download.Download(url,
		download.WithDestDir(binDir),
		download.WithFormat("tar.gz"),
		download.WithExtract(download.WithExtractFile(binaryName)),
		download.WithSymlink(),
		download.WithSkipIfExists(binaryPath),
	)
}
