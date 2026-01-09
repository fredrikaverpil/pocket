// Package golangcilint provides golangci-lint tool integration.
package golangcilint

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tool"
)

const name = "golangci-lint"

// renovate: datasource=github-releases depName=golangci/golangci-lint
const version = "2.7.1"

//go:embed golangci.yml
var defaultConfig []byte

var t = &tool.Tool{Name: name, Prepare: Prepare}

// Command prepares the tool and returns an exec.Cmd for running golangci-lint.
var Command = t.Command

// Run installs (if needed) and executes golangci-lint.
var Run = t.Run

// ConfigPath returns the path to the golangci-lint config file.
// It checks for .golangci.yml in the repo root first, then falls back
// to the bundled default config.
func ConfigPath() (string, error) {
	// Check for user config in repo root
	repoConfig := pocket.FromGitRoot(".golangci.yml")
	if _, err := os.Stat(repoConfig); err == nil {
		return repoConfig, nil
	}

	// Also check for .golangci.yaml
	repoConfigYaml := pocket.FromGitRoot(".golangci.yaml")
	if _, err := os.Stat(repoConfigYaml); err == nil {
		return repoConfigYaml, nil
	}

	// Write bundled config to .pocket/tools/golangci-lint/golangci.yml
	configDir := pocket.FromToolsDir(name)
	configPath := filepath.Join(configDir, "golangci.yml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			return "", fmt.Errorf("create config dir: %w", err)
		}
		if err := os.WriteFile(configPath, defaultConfig, 0o644); err != nil {
			return "", fmt.Errorf("write default config: %w", err)
		}
	}

	return configPath, nil
}

// Prepare ensures golangci-lint is installed.
func Prepare(ctx context.Context) error {
	binDir := pocket.FromToolsDir(name, version, "bin")
	binaryName := pocket.BinaryName(name)
	binary := filepath.Join(binDir, binaryName)

	// Windows uses .zip, others use .tar.gz.
	var binURL string
	var opts []tool.Opt
	if runtime.GOOS == "windows" {
		binURL = fmt.Sprintf(
			"https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.zip",
			version,
			version,
			runtime.GOOS,
			archName(),
		)
		opts = []tool.Opt{
			tool.WithDestinationDir(binDir),
			tool.WithUnzip(),
			tool.WithExtractFiles(binaryName),
			tool.WithSkipIfFileExists(binary),
			tool.WithSymlink(binary),
		}
	} else {
		binURL = fmt.Sprintf(
			"https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.tar.gz",
			version,
			version,
			runtime.GOOS,
			archName(),
		)
		opts = []tool.Opt{
			tool.WithDestinationDir(binDir),
			tool.WithUntarGz(),
			tool.WithExtractFiles(binaryName),
			tool.WithSkipIfFileExists(binary),
			tool.WithSymlink(binary),
		}
	}

	return tool.FromRemote(ctx, binURL, opts...)
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
