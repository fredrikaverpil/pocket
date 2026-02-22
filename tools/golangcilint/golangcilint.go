// Package golangcilint provides the golangci-lint tool for Go linting.
package golangcilint

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/golang"
)

// Name is the binary name for golangci-lint.
const Name = "golangci-lint"

// Version is the version of golangci-lint to install.
// renovate: datasource=go depName=github.com/golangci/golangci-lint/v2
const Version = "v2.1.6"

//go:embed golangci.yml
var defaultConfig []byte

// DefaultConfigFile is the name of the default config file.
const DefaultConfigFile = ".golangci.yml"

// configFileNames are the filenames golangci-lint searches for.
var configFileNames = []string{
	".golangci.yml",
	".golangci.yaml",
	".golangci.toml",
	".golangci.json",
}

// EnsureDefaultConfig writes the bundled config to .pocket/tools/golangci-lint/
// and returns its path. Safe to call multiple times.
func EnsureDefaultConfig() string {
	configPath := pk.FromToolsDir("golangci-lint", DefaultConfigFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		_ = os.MkdirAll(filepath.Dir(configPath), 0o755)
		_ = os.WriteFile(configPath, defaultConfig, 0o644)
	}
	return configPath
}

// HasProjectConfig checks if the project has its own golangci-lint config file
// at the git root.
func HasProjectConfig() bool {
	for _, name := range configFileNames {
		if _, err := os.Stat(pk.FromGitRoot(name)); err == nil {
			return true
		}
	}
	return false
}

// Install is a hidden, global task that installs golangci-lint.
// Global ensures it only runs once regardless of path context.
var Install = &pk.Task{
	Name:   "install:golangci-lint",
	Usage:  "install golangci-lint",
	Body:   golang.Install("github.com/golangci/golangci-lint/v2/cmd/golangci-lint", Version),
	Hidden: true,
	Global: true,
}
