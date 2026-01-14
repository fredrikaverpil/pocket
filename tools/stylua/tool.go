// Package stylua provides stylua tool integration.
package stylua

import (
	"context"
	_ "embed"
	"fmt"
	"path/filepath"

	"github.com/fredrikaverpil/pocket"
)

const name = "stylua"

// renovate: datasource=github-releases depName=JohnnyMorganz/StyLua
const version = "2.3.1"

//go:embed stylua.toml
var defaultConfig []byte

// Tool is the stylua tool.
//
// Example usage in a task action:
//
//	configPath, _ := stylua.Tool.ConfigPath()
//	stylua.Tool.Exec(ctx, tc, "-f", configPath, ".")
var Tool = pocket.NewTool(name, version, install).
	WithConfig(pocket.ToolConfig{
		UserFiles:   []string{"stylua.toml", ".stylua.toml"},
		DefaultFile: "stylua.toml",
		DefaultData: defaultConfig,
	})

func install(ctx context.Context, tc *pocket.TaskContext) error {
	binDir := pocket.FromToolsDir(name, version, "bin")
	binaryName := pocket.BinaryName(name)
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
		version, hostOS, hostArch,
	)

	return pocket.Download(ctx, tc, url,
		pocket.WithDestDir(binDir),
		pocket.WithFormat("zip"),
		pocket.WithExtract(pocket.WithExtractFile(binaryName)),
		pocket.WithSymlink(),
		pocket.WithSkipIfExists(binaryPath),
	)
}
