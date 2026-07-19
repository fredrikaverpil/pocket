// Package rumdl provides rumdl (Markdown linter and formatter) integration.
package rumdl

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/download"
	"github.com/fredrikaverpil/pocket/pk/platform"
	"github.com/fredrikaverpil/pocket/pk/repopath"
)

// Name is the binary name for rumdl.
const Name = "rumdl"

// Version is the version of rumdl to install.
// renovate: datasource=github-releases depName=rvben/rumdl
const Version = "0.2.36"

//go:embed rumdl.toml
var defaultConfig []byte

// DefaultConfigFile is the name of the default config file.
const DefaultConfigFile = "rumdl.toml"

// configFileNames are the filenames rumdl searches for.
// Note: pyproject.toml with a [tool.rumdl] section is also supported by rumdl
// but not checked for here.
var configFileNames = []string{
	".rumdl.toml",
	"rumdl.toml",
	".config/rumdl.toml",
}

// EnsureDefaultConfig writes the bundled config to .pocket/tools/rumdl/
// and returns its path. The file is rewritten on every call so it stays in
// sync with the embedded config. Safe to call multiple times.
func EnsureDefaultConfig() string {
	configPath := repopath.FromToolsDir("rumdl", DefaultConfigFile)
	_ = os.MkdirAll(filepath.Dir(configPath), 0o755)
	_ = os.WriteFile(configPath, defaultConfig, 0o644)
	return configPath
}

// HasProjectConfig checks if the project has its own rumdl config file
// at the git root.
func HasProjectConfig() bool {
	for _, name := range configFileNames {
		if _, err := os.Stat(repopath.FromGitRoot(name)); err == nil {
			return true
		}
	}
	return false
}

// Install ensures rumdl is available.
var Install = &pk.Task{
	Name:   "install:rumdl",
	Usage:  "install rumdl",
	Body:   installRumdl(),
	Hidden: true,
	Global: true,
}

func installRumdl() pk.Runnable {
	binDir := repopath.FromToolsDir("rumdl", Version, "bin")
	binaryName := platform.BinaryName("rumdl")
	binaryPath := filepath.Join(binDir, binaryName)

	// rumdl release assets: rumdl-v<version>-<target>.<tar.gz|zip>
	target := platform.ArchToX8664(platform.HostArch()) // amd64→x86_64, arm64→aarch64
	switch platform.HostOS() {
	case platform.Darwin:
		target += "-apple-darwin"
	case platform.Linux:
		target += "-unknown-linux-gnu"
	case platform.Windows:
		target = "x86_64-pc-windows-msvc" // only x86_64 is published for Windows
	}

	format := platform.DefaultArchiveFormat() // "zip" on Windows, "tar.gz" otherwise
	url := fmt.Sprintf(
		"https://github.com/rvben/rumdl/releases/download/v%s/rumdl-v%s-%s.%s",
		Version, Version, target, format,
	)

	return download.Download(url,
		download.WithDestDir(binDir),
		download.WithFormat(format),
		download.WithExtract(download.WithExtractFile(binaryName)),
		download.WithSymlink(),
		download.WithSkipIfExists(binaryPath),
	)
}
