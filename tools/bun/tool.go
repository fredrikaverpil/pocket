// Package bun provides bun (JavaScript runtime & package manager) tool integration.
package bun

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/fredrikaverpil/pocket"
)

const name = "bun"

// renovate: datasource=github-releases depName=oven-sh/bun extractVersion=^bun-v(?<version>.*)$
const version = "1.3.6"

// Tool is the bun tool.
//
// Example usage in a task action:
//
//	bun.Tool.Exec(ctx, tc, "run", "script")
var Tool = pocket.NewTool(name, version, install)

// Install installs an npm package using bun into the specified directory.
// The package should include version, e.g., "prettier@3.7.4".
func Install(ctx context.Context, tc *pocket.TaskContext, installDir, pkg string) error {
	return Tool.Exec(ctx, tc, "install", "--cwd", installDir, pkg)
}

// BinaryPath returns the path to a binary installed by bun in the given directory.
// On Windows, it appends .exe to the binary name.
func BinaryPath(installDir, binaryName string) string {
	return filepath.Join(installDir, "node_modules", ".bin", pocket.BinaryName(binaryName))
}

func install(ctx context.Context, tc *pocket.TaskContext) error {
	binDir := pocket.FromToolsDir(name, version, "bin")
	binaryName := pocket.BinaryName(name)
	binaryPath := filepath.Join(binDir, binaryName)

	url := fmt.Sprintf(
		"https://github.com/oven-sh/bun/releases/download/bun-v%s/bun-%s.zip",
		version,
		platformArch(),
	)

	return pocket.Download(ctx, tc, url,
		pocket.WithDestDir(binDir),
		pocket.WithFormat("zip"),
		pocket.WithExtract(pocket.WithExtractFile(binaryName)),
		pocket.WithSymlink(),
		pocket.WithSkipIfExists(binaryPath),
	)
}

func platformArch() string {
	switch runtime.GOOS {
	case pocket.Darwin:
		if runtime.GOARCH == pocket.ARM64 {
			return "darwin-aarch64"
		}
		return "darwin-x64"
	case pocket.Linux:
		if runtime.GOARCH == pocket.ARM64 {
			return "linux-aarch64"
		}
		return "linux-x64"
	case pocket.Windows:
		return "windows-x64"
	default:
		return fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	}
}
