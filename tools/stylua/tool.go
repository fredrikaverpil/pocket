// Package stylua provides stylua tool integration.
package stylua

import (
	"context"
	_ "embed"
	"fmt"
	"runtime"

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

	url := fmt.Sprintf(
		"https://github.com/JohnnyMorganz/StyLua/releases/download/v%s/stylua-%s-%s.zip",
		version,
		osName(),
		archName(),
	)

	return pocket.DownloadBinary(ctx, tc, url, pocket.DownloadOpts{
		DestDir:      binDir,
		Format:       "zip",
		ExtractFiles: []string{binaryName},
		Symlink:      true,
	})
}

func osName() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	case "linux":
		return "linux"
	case "windows":
		return "windows"
	default:
		return runtime.GOOS
	}
}

func archName() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		return runtime.GOARCH
	}
}
