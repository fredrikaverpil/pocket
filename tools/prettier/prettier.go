// Package prettier provides prettier (code formatter) integration.
// prettier is installed via bun into a local directory.
package prettier

import (
	"context"
	_ "embed"
	"os"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/bun"
)

// Name is the binary name for prettier.
const Name = "prettier"

// renovate: datasource=npm depName=prettier
const Version = "3.7.4"

//go:embed prettierrc.json
var defaultConfig []byte

//go:embed prettierignore
var defaultIgnore []byte

// Install ensures prettier is available.
var Install = pocket.Func("install:prettier", "install prettier", install).Hidden()

func install(ctx context.Context) error {
	installDir := pocket.FromToolsDir(Name, Version)
	binary := bun.BinaryPath(installDir, Name)

	// Skip if already installed.
	if _, err := os.Stat(binary); err == nil {
		_, err := pocket.CreateSymlink(binary)
		return err
	}

	pocket.Printf(ctx, "Installing prettier %s...\n", Version)

	// Ensure bun is installed.
	pocket.Serial(ctx, bun.Install)

	// Create install directory.
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}

	// Install prettier using bun.
	if err := pocket.Exec(ctx, bun.Name, "install", "--cwd", installDir, Name+"@"+Version); err != nil {
		return err
	}

	// Create symlink to .pocket/bin/.
	_, err := pocket.CreateSymlink(binary)
	return err
}

// Config for prettier configuration file lookup.
var Config = pocket.ToolConfig{
	UserFiles: []string{
		".prettierrc",
		".prettierrc.json",
		".prettierrc.yaml",
		".prettierrc.yml",
		"prettier.config.js",
		"prettier.config.mjs",
	},
	DefaultFile: ".prettierrc",
	DefaultData: defaultConfig,
}

// DefaultIgnore returns the default .prettierignore content.
func DefaultIgnore() []byte {
	return defaultIgnore
}

// EnsureIgnoreFile ensures a .prettierignore file exists at git root.
func EnsureIgnoreFile() (string, error) {
	ignoreFile := pocket.FromGitRoot(".prettierignore")

	if _, err := os.Stat(ignoreFile); err == nil {
		return ignoreFile, nil
	}

	if err := os.WriteFile(ignoreFile, defaultIgnore, 0o644); err != nil {
		return "", err
	}
	return ignoreFile, nil
}
