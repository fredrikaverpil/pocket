// Package tsqueryls provides the ts_query_ls tool for tree-sitter query files.
package tsqueryls

import (
	"fmt"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/download"
	"github.com/fredrikaverpil/pocket/pk/platform"
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
	binaryName := platform.BinaryName(Name)
	binaryPath := filepath.Join(binDir, binaryName)

	url, format := buildDownloadURL()

	return download.Download(url,
		download.WithDestDir(binDir),
		download.WithFormat(format),
		download.WithExtract(download.WithExtractFile(binaryName)),
		download.WithSymlink(),
		download.WithSkipIfExists(binaryPath),
	)
}

func buildDownloadURL() (url, format string) {
	hostOS := platform.HostOS()
	hostArch := platform.HostArch()

	// Build platform suffix matching ts_query_ls naming convention.
	var plat string
	switch {
	case hostOS == platform.Darwin && hostArch == platform.ARM64:
		plat = "aarch64-apple-darwin"
	case hostOS == platform.Darwin && hostArch == platform.AMD64:
		plat = "x86_64-apple-darwin"
	case hostOS == platform.Linux && hostArch == platform.ARM64:
		plat = "aarch64-unknown-linux-gnu"
	case hostOS == platform.Linux && hostArch == platform.AMD64:
		plat = "x86_64-unknown-linux-gnu"
	case hostOS == platform.Windows && hostArch == platform.AMD64:
		plat = "x86_64-pc-windows-msvc"
	default:
		// Fallback - will likely fail but gives a useful error.
		plat = fmt.Sprintf("%s-%s", hostArch, hostOS)
	}

	// Windows uses zip, others use tar.gz.
	ext := "tar.gz"
	format = "tar.gz"
	if hostOS == platform.Windows {
		ext = "zip"
		format = "zip"
	}

	url = fmt.Sprintf(
		"https://github.com/ribru17/ts_query_ls/releases/download/%s/ts_query_ls-%s.%s",
		Version, plat, ext,
	)

	return url, format
}
