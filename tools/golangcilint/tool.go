// Package golangcilint provides golangci-lint tool integration.
package golangcilint

import (
	"context"
	_ "embed"
	"fmt"
	"runtime"

	"github.com/fredrikaverpil/pocket"
)

const name = "golangci-lint"

// renovate: datasource=github-releases depName=golangci/golangci-lint
const version = "2.7.1"

//go:embed golangci.yml
var defaultConfig []byte

// Tool is the golangci-lint tool.
//
// Example usage in a task action:
//
//	configPath, _ := golangcilint.Tool.ConfigPath()
//	golangcilint.Tool.Exec(ctx, tc, "run", "-c", configPath, "./...")
var Tool = pocket.NewTool(name, version, install).
	WithConfig(pocket.ToolConfig{
		UserFiles:   []string{".golangci.yml", ".golangci.yaml"},
		DefaultFile: "golangci.yml",
		DefaultData: defaultConfig,
	})

func install(ctx context.Context, tc *pocket.TaskContext) error {
	binDir := pocket.FromToolsDir(name, version, "bin")
	binaryName := pocket.BinaryName(name)

	var format string
	if runtime.GOOS == "windows" {
		format = "zip"
	} else {
		format = "tar.gz"
	}

	url := fmt.Sprintf(
		"https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.%s",
		version,
		version,
		runtime.GOOS,
		archName(),
		format,
	)

	return pocket.DownloadBinary(ctx, tc, url, pocket.DownloadOpts{
		DestDir:      binDir,
		Format:       format,
		ExtractFiles: []string{binaryName},
		Symlink:      true,
	})
}

func archName() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return runtime.GOARCH
	}
}
