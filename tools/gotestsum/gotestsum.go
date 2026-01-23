// Package gotestsum provides gotestsum CLI tool integration.
package gotestsum

import (
	"fmt"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
)

// Name is the binary name for gotestsum.
const Name = "gotestsum"

// Version is the version of gotestsum to install.
// renovate: datasource=github-releases depName=gotestyourself/gotestsum
const Version = "1.13.0"

// Install ensures gotestsum CLI is available.
var Install = pk.NewTask("install:gotestsum", "install gotestsum CLI", nil,
	installGotestsum(),
).Hidden().Global()

func installGotestsum() pk.Runnable {
	binDir := pk.FromToolsDir("gotestsum", Version, "bin")
	binaryName := pk.BinaryName(Name)
	binaryPath := filepath.Join(binDir, binaryName)

	// Asset name: gotestsum_{version}_{os}_{arch}.tar.gz
	assetName := fmt.Sprintf("gotestsum_%s_%s_%s.tar.gz", Version, pk.HostOS(), pk.HostArch())
	url := fmt.Sprintf(
		"https://github.com/gotestyourself/gotestsum/releases/download/v%s/%s",
		Version, assetName,
	)

	return pk.Download(url,
		pk.WithDestDir(binDir),
		pk.WithFormat("tar.gz"),
		pk.WithExtract(pk.WithRenameFile(Name, binaryName)),
		pk.WithSymlink(),
		pk.WithSkipIfExists(binaryPath),
	)
}
