// Package golangcilint provides golangci-lint tool integration.
package golangcilint

import (
	"context"
	_ "embed"
	"fmt"
	"path/filepath"

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
	binaryPath := filepath.Join(binDir, binaryName)

	hostOS := pocket.HostOS()
	hostArch := pocket.HostArch()
	format := pocket.DefaultArchiveFormat()

	url := fmt.Sprintf(
		"https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.%s",
		version, version, hostOS, hostArch, format,
	)

	return pocket.Download(ctx, tc, url,
		pocket.WithDestDir(binDir),
		pocket.WithFormat(format),
		pocket.WithExtract(pocket.WithExtractFile(binaryName)),
		pocket.WithSymlink(),
		pocket.WithSkipIfExists(binaryPath),
	)
}
