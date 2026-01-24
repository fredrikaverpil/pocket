// Package tsqueryls provides the ts_query_ls tool for tree-sitter query files.
package tsqueryls

import (
	"fmt"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
)

// Name is the binary name for ts_query_ls.
const Name = "ts_query_ls"

// Version is the ts_query_ls version to install.
// renovate: datasource=github-releases depName=ribru17/ts_query_ls
const Version = "v3.15.1"

// Install creates a task that ensures ts_query_ls is available.
// ts_query_ls is used for formatting and linting tree-sitter query (.scm) files.
var Install = pk.NewTask("install:ts_query_ls", "install ts_query_ls", nil,
	installTSQueryLs(),
).Hidden().Global()

func installTSQueryLs() pk.Runnable {
	binDir := pk.FromToolsDir("tsqueryls", Version, "bin")
	binaryName := pk.BinaryName(Name)
	binaryPath := filepath.Join(binDir, binaryName)

	url, format := buildDownloadURL()

	return pk.Download(url,
		pk.WithDestDir(binDir),
		pk.WithFormat(format),
		pk.WithExtract(pk.WithExtractFile(binaryName)),
		pk.WithSymlink(),
		pk.WithSkipIfExists(binaryPath),
	)
}

func buildDownloadURL() (url, format string) {
	hostOS := pk.HostOS()
	hostArch := pk.HostArch()

	// Build platform suffix matching ts_query_ls naming convention.
	var platform string
	switch {
	case hostOS == pk.Darwin && hostArch == pk.ARM64:
		platform = "aarch64-apple-darwin"
	case hostOS == pk.Darwin && hostArch == pk.AMD64:
		platform = "x86_64-apple-darwin"
	case hostOS == pk.Linux && hostArch == pk.ARM64:
		platform = "aarch64-unknown-linux-gnu"
	case hostOS == pk.Linux && hostArch == pk.AMD64:
		platform = "x86_64-unknown-linux-gnu"
	case hostOS == pk.Windows && hostArch == pk.AMD64:
		platform = "x86_64-pc-windows-msvc"
	default:
		// Fallback - will likely fail but gives a useful error.
		platform = fmt.Sprintf("%s-%s", hostArch, hostOS)
	}

	// Windows uses zip, others use tar.gz.
	ext := "tar.gz"
	format = "tar.gz"
	if hostOS == pk.Windows {
		ext = "zip"
		format = "zip"
	}

	url = fmt.Sprintf(
		"https://github.com/ribru17/ts_query_ls/releases/download/%s/ts_query_ls-%s.%s",
		Version, platform, ext,
	)

	return url, format
}
