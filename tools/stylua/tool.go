// Package stylua provides stylua tool integration.
package stylua

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

const name = "stylua"

// renovate: datasource=github-releases depName=JohnnyMorganz/StyLua
const version = "2.3.1"

//go:embed stylua.toml
var defaultConfig []byte

var t = &tool.Tool{Name: name, Prepare: Prepare}

// Command prepares the tool and returns an exec.Cmd for running stylua.
var Command = t.Command

// Run installs (if needed) and executes stylua.
var Run = t.Run

// ConfigPath returns the path to the stylua config file.
// It checks for stylua.toml in the repo root first, then falls back
// to the bundled default config.
func ConfigPath() (string, error) {
	// Check for user config in repo root
	repoConfig := pocket.FromGitRoot("stylua.toml")
	if _, err := os.Stat(repoConfig); err == nil {
		return repoConfig, nil
	}

	// Write bundled config to .pocket/tools/stylua/stylua.toml
	configDir := pocket.FromToolsDir(name)
	configPath := filepath.Join(configDir, "stylua.toml")

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

// Prepare ensures stylua is installed.
func Prepare(ctx context.Context) error {
	binDir := pocket.FromToolsDir(name, version, "bin")
	binaryName := pocket.BinaryName(name)
	binary := filepath.Join(binDir, binaryName)

	binURL := fmt.Sprintf(
		"https://github.com/JohnnyMorganz/StyLua/releases/download/v%s/stylua-%s-%s.zip",
		version,
		osName(),
		archName(),
	)

	return tool.FromRemote(
		ctx,
		binURL,
		tool.WithDestinationDir(binDir),
		tool.WithUnzip(),
		tool.WithSkipIfFileExists(binary),
		tool.WithSymlink(binary),
	)
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
