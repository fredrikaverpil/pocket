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
	"github.com/goyek/goyek/v3"
)

const name = "golangci-lint"

// renovate: datasource=github-releases depName=golangci/golangci-lint
const version = "2.7.1"

//go:embed golangci.yml
var defaultConfig []byte

// Prepare is a goyek task that downloads and installs golangci-lint.
// Hidden from task list (no Usage field).
var Prepare = goyek.Define(goyek.Task{
	Name: "golangci-lint:prepare",
	Action: func(a *goyek.A) {
		if err := prepare(a.Context()); err != nil {
			a.Fatal(err)
		}
	},
})

// Command returns an exec.Cmd for running golangci-lint.
// Call Prepare first or use as a goyek Deps.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	return bld.Command(ctx, bld.FromBinDir(name), args...)
}

// Run executes golangci-lint with the given arguments.
// Call Prepare first or use as a goyek Deps.
func Run(ctx context.Context, args ...string) error {
	return Command(ctx, args...).Run()
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

func prepare(ctx context.Context) error {
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
