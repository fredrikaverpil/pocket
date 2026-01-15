// Package prettier provides prettier (code formatter) integration.
// prettier is installed via bun into a local directory with locked dependencies.
package prettier

import (
	"context"
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/fredrikaverpil/pocket"
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

var (
	versionOnce sync.Once
	version     string
)

// Version returns the prettier version from package.json.
func Version() string {
	versionOnce.Do(func() {
		var pkg struct {
			Dependencies map[string]string `json:"dependencies"`
		}
		if err := json.Unmarshal(packageJSON, &pkg); err == nil {
			version = pkg.Dependencies[Name]
		}
	})
	return version
}

// Install ensures prettier is available.
//
// To update prettier version:
//  1. Update version in package.json
//  2. cd tools/prettier && bun install && rm -rf node_modules
//  3. git add package.json bun.lock
var Install = pocket.Func("install:prettier", "install prettier", pocket.Serial(
	bun.Install,
	install,
)).Hidden()

func install(ctx context.Context) error {
	installDir := pocket.FromToolsDir(Name, Version())
	binary := bun.BinaryPath(installDir, Name)

	// Skip if already installed.
	if _, err := os.Stat(binary); err == nil {
		_, err := pocket.CreateSymlink(binary)
		return err
	}

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
	if err := bun.InstallFromLockfile(ctx, installDir); err != nil {
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

// Exec runs prettier with the given arguments.
// Uses bun.Run internally for cross-platform reliability (avoids Windows shim issues).
func Exec(ctx context.Context, args ...string) error {
	installDir := pocket.FromToolsDir(Name, Version())
	return bun.Run(ctx, installDir, Name, args...)
}
