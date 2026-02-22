// Package prettier provides prettier (code formatter) integration.
// prettier is installed via bun into a local directory with locked dependencies.
package prettier

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/bun"
)

// Name is the binary name for prettier.
const Name = "prettier"

//go:embed prettierrc.json
var defaultConfig []byte

//go:embed prettierignore
var defaultIgnore []byte

//go:embed package.json
var packageJSON []byte

//go:embed bun.lock
var lockfile []byte

// Version returns a content hash based on package.json and bun.lock.
func Version() string {
	return bun.ContentHash(packageJSON, lockfile)
}

// Install ensures prettier is available.
var Install = &pk.Task{
	Name:   "install:prettier",
	Usage:  "install prettier",
	Body:   pk.Serial(bun.Install, installPrettier()),
	Hidden: true,
	Global: true,
}

func installPrettier() pk.Runnable {
	installDir := pk.FromToolsDir(Name, Version())
	return bun.EnsureInstalled(installDir, Name, func(ctx context.Context) error {
		// Create install directory and write lockfile.
		if err := os.MkdirAll(installDir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(installDir, "package.json"), packageJSON, 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(installDir, "bun.lock"), lockfile, 0o644); err != nil {
			return err
		}

		// Install prettier using bun with frozen lockfile.
		return bun.InstallFromLockfile(ctx, installDir)
	})
}

// DefaultConfigFile is the name of the default config file.
const DefaultConfigFile = ".prettierrc"

// configFileNames are the filenames prettier searches for.
var configFileNames = []string{
	".prettierrc",
	".prettierrc.json",
	".prettierrc.yml",
	".prettierrc.yaml",
	".prettierrc.toml",
	".prettierrc.js",
	".prettierrc.cjs",
	".prettierrc.mjs",
	"prettier.config.js",
	"prettier.config.cjs",
	"prettier.config.mjs",
}

// EnsureDefaultConfig writes the bundled config to .pocket/tools/prettier/
// and returns its path. Safe to call multiple times.
func EnsureDefaultConfig() string {
	configPath := pk.FromToolsDir("prettier", DefaultConfigFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		_ = os.MkdirAll(filepath.Dir(configPath), 0o755)
		_ = os.WriteFile(configPath, defaultConfig, 0o644)
	}
	return configPath
}

// HasProjectConfig checks if the project has its own prettier config file
// at the git root.
func HasProjectConfig() bool {
	for _, name := range configFileNames {
		if _, err := os.Stat(pk.FromGitRoot(name)); err == nil {
			return true
		}
	}
	return false
}

// EnsureIgnoreFile ensures a .prettierignore file exists at git root.
func EnsureIgnoreFile() (string, error) {
	ignoreFile := pk.FromGitRoot(".prettierignore")

	if _, err := os.Stat(ignoreFile); err == nil {
		return ignoreFile, nil
	}

	if err := os.WriteFile(ignoreFile, defaultIgnore, 0o644); err != nil {
		return "", err
	}
	return ignoreFile, nil
}

// Exec runs prettier with the given arguments.
func Exec(ctx context.Context, args ...string) error {
	installDir := pk.FromToolsDir(Name, Version())
	// Run via bun since prettier is a Node.js script (shebang: #!/usr/bin/env node).
	return bun.Run(ctx, installDir, Name, args...)
}
