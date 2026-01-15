// Package stylua provides stylua tool integration.
package stylua

import (
	"context"
	_ "embed"
	"fmt"
	"path/filepath"

	"github.com/fredrikaverpil/pocket"
)

// Name is the binary name for stylua.
const Name = "stylua"

// renovate: datasource=github-releases depName=JohnnyMorganz/StyLua
const Version = "2.3.1"

//go:embed stylua.toml
var defaultConfig []byte

// Config describes how to find or create stylua's configuration file.
var Config = pocket.ToolConfig{
	UserFiles:   []string{"stylua.toml", ".stylua.toml"},
	DefaultFile: "stylua.toml",
	DefaultData: defaultConfig,
}

// Install ensures stylua is available.
var Install = pocket.Func("install:stylua", "install stylua", install).Hidden()

func install(ctx context.Context) error {
	binDir := pocket.FromToolsDir("stylua", Version, "bin")
	binaryName := pocket.BinaryName("stylua")
	binaryPath := filepath.Join(binDir, binaryName)

	hostOS := pocket.HostOS()
	hostArch := pocket.HostArch()

	// StyLua uses different naming: darwin->macos, amd64->x86_64, arm64->aarch64
	if hostOS == pocket.Darwin {
		hostOS = "macos"
	}
	hostArch = pocket.ArchToX8664(hostArch)

	url := fmt.Sprintf(
		"https://github.com/JohnnyMorganz/StyLua/releases/download/v%s/stylua-%s-%s.zip",
		Version, hostOS, hostArch,
	)

	return pocket.Download(ctx, url,
		pocket.WithDestDir(binDir),
		pocket.WithFormat("zip"),
		pocket.WithExtract(pocket.WithExtractFile(binaryName)),
		pocket.WithSymlink(),
		pocket.WithSkipIfExists(binaryPath),
	)
}
