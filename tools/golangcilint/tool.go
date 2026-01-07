// Package golangcilint provides golangci-lint tool integration.
package golangcilint

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/fredrikaverpil/bld"
	"github.com/fredrikaverpil/bld/tool"
)

const name = "golangci-lint"

// renovate: datasource=github-releases depName=golangci/golangci-lint
const version = "2.7.1"

//go:embed golangci.yml
var defaultConfig []byte

// Command prepares the tool and returns an exec.Cmd for running golangci-lint.
func Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
	if err := Prepare(ctx); err != nil {
		return nil, err
	}
	return bld.Command(ctx, bld.FromBinDir(name), args...), nil
}

// Run installs (if needed) and executes golangci-lint.
func Run(ctx context.Context, args ...string) error {
	cmd, err := Command(ctx, args...)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// ConfigPath returns the path to the golangci-lint config file.
// It checks for .golangci.yml in the repo root first, then falls back
// to the bundled default config.
func ConfigPath() (string, error) {
	// Check for user config in repo root
	repoConfig := bld.FromGitRoot(".golangci.yml")
	if _, err := os.Stat(repoConfig); err == nil {
		return repoConfig, nil
	}

	// Also check for .golangci.yaml
	repoConfigYaml := bld.FromGitRoot(".golangci.yaml")
	if _, err := os.Stat(repoConfigYaml); err == nil {
		return repoConfigYaml, nil
	}

	// Write bundled config to .bld/tools/golangci-lint/golangci.yml
	configDir := bld.FromToolsDir(name)
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
	binDir := bld.FromToolsDir(name, version, "bin")
	binary := filepath.Join(binDir, name)

	binURL := fmt.Sprintf(
		"https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.tar.gz",
		version,
		version,
		runtime.GOOS,
		archName(),
	)

	return tool.FromRemote(
		ctx,
		binURL,
		tool.WithDestinationDir(binDir),
		tool.WithUntarGz(),
		tool.WithExtractFiles(name),
		tool.WithSkipIfFileExists(binary),
		tool.WithSymlink(binary),
	)
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
