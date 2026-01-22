// Package stylua provides stylua tool integration.
package stylua

import (
	_ "embed"
	"fmt"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
)

// Name is the binary name for stylua.
const Name = "stylua"

// Version is the version of stylua to install.
// renovate: datasource=github-releases depName=JohnnyMorganz/StyLua
const Version = "2.3.1"

//go:embed stylua.toml
var defaultConfig []byte

// Config describes how to find or create stylua's configuration file.
var Config = pk.ToolConfig{
	UserFiles:   []string{"stylua.toml", ".stylua.toml"},
	DefaultFile: "stylua.toml",
	DefaultData: defaultConfig,
}

// Install ensures stylua is available.
var Install = pk.NewTask("install:stylua", "install stylua", nil,
	installStylua(),
).Hidden()

func installStylua() pk.Runnable {
	binDir := pk.FromToolsDir("stylua", Version, "bin")
	binaryName := pk.BinaryName("stylua")
	binaryPath := filepath.Join(binDir, binaryName)

	hostOS := pk.HostOS()
	hostArch := pk.HostArch()

	// StyLua uses different naming: darwin->macos, amd64->x86_64, arm64->aarch64
	if hostOS == pk.Darwin {
		hostOS = "macos"
	}
	hostArch = pk.ArchToX8664(hostArch)

	url := fmt.Sprintf(
		"https://github.com/JohnnyMorganz/StyLua/releases/download/v%s/stylua-%s-%s.zip",
		Version, hostOS, hostArch,
	)

	return pk.Download(url,
		pk.WithDestDir(binDir),
		pk.WithFormat("zip"),
		pk.WithExtract(pk.WithExtractFile(binaryName)),
		pk.WithSymlink(),
		pk.WithSkipIfExists(binaryPath),
	)
}
