// Package tsqueryls provides ts_query_ls tool integration.
// ts_query_ls is a tree-sitter query file formatter and linter.
package tsqueryls

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/fredrikaverpil/pocket"
)

// Name is the binary name for ts_query_ls.
const Name = "ts_query_ls"

// renovate: datasource=github-releases depName=ribru17/ts_query_ls
const Version = "3.15.1"

// Install ensures ts_query_ls is available.
var Install = pocket.Func("install:ts_query_ls", "install ts_query_ls", install).Hidden()

func install(ctx context.Context) error {
	binDir := pocket.FromToolsDir("ts_query_ls", Version, "bin")
	binaryName := pocket.BinaryName("ts_query_ls")
	binaryPath := filepath.Join(binDir, binaryName)

	platform := platformArch()
	ext := "tar.gz"
	if runtime.GOOS == pocket.Windows {
		ext = "zip"
	}

	url := fmt.Sprintf(
		"https://github.com/ribru17/ts_query_ls/releases/download/v%s/ts_query_ls-%s.%s",
		Version, platform, ext,
	)

	return pocket.Download(ctx, url,
		pocket.WithDestDir(binDir),
		pocket.WithFormat(ext),
		pocket.WithExtract(pocket.WithExtractFile(binaryName)),
		pocket.WithSymlink(),
		pocket.WithSkipIfExists(binaryPath),
	)
}

func platformArch() string {
	switch runtime.GOOS {
	case pocket.Darwin:
		if runtime.GOARCH == pocket.ARM64 {
			return "aarch64-apple-darwin"
		}
		return "x86_64-apple-darwin"
	case pocket.Linux:
		if runtime.GOARCH == pocket.ARM64 {
			return "aarch64-unknown-linux-gnu"
		}
		return "x86_64-unknown-linux-gnu"
	case pocket.Windows:
		return "x86_64-pc-windows-msvc"
	default:
		return fmt.Sprintf("%s-%s", runtime.GOARCH, runtime.GOOS)
	}
}
