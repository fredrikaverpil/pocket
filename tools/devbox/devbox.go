// Package devbox provides devbox tool integration.
// Devbox creates portable, isolated dev environments using Nix.
//
// The devbox binary is downloaded from GitHub releases and installed
// into .pocket/tools/devbox/{version}/bin/devbox. Packages defined
// in devbox.json are managed by devbox itself (in /nix/store and
// .devbox/), not in .pocket/tools/.
package devbox

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/download"
)

// Name is the binary name for devbox.
const Name = "devbox"

// Version is the version of devbox to install.
// renovate: datasource=github-releases depName=jetify-com/devbox
const Version = "0.16.0"

// Install ensures devbox is available.
var Install = &pk.Task{
	Name:   "install:devbox",
	Usage:  "install devbox",
	Body:   installDevbox(),
	Hidden: true,
	Global: true,
}

func installDevbox() pk.Runnable {
	binDir := pk.FromToolsDir(Name, Version, "bin")
	binaryName := pk.BinaryName(Name)
	binaryPath := filepath.Join(binDir, binaryName)

	url := fmt.Sprintf(
		"https://github.com/jetify-com/devbox/releases/download/%s/devbox_%s_%s_%s.tar.gz",
		Version, Version, platformOS(), platformArch(),
	)

	return download.Download(url,
		download.WithDestDir(binDir),
		download.WithFormat("tar.gz"),
		download.WithExtract(download.WithExtractFile(binaryName)),
		download.WithSymlink(),
		download.WithSkipIfExists(binaryPath),
	)
}

// Exec runs a command inside the devbox environment.
// This uses "devbox run --" which sets up the Nix-managed PATH
// and environment variables before executing the command.
func Exec(ctx context.Context, name string, args ...string) error {
	runArgs := []string{"run", "--", name}
	runArgs = append(runArgs, args...)
	return pk.Exec(ctx, Name, runArgs...)
}

func platformOS() string {
	return runtime.GOOS
}

func platformArch() string {
	switch runtime.GOARCH {
	case pk.AMD64:
		return pk.AMD64
	case pk.ARM64:
		return pk.ARM64
	default:
		return runtime.GOARCH
	}
}
