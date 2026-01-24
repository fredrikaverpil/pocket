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
var Install = pk.NewTask("install:prettier", "install prettier", nil,
	pk.Serial(bun.Install, installPrettier()),
).Hidden().Global()

func installPrettier() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		installDir := pk.FromToolsDir(Name, Version())
		binary := bun.BinaryPath(installDir, Name)

		// Skip if already installed.
		if _, err := os.Stat(binary); err == nil {
			return nil
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
		// No symlink needed since Exec() runs via bun.Run().
		return bun.InstallFromLockfile(ctx, installDir)
	})
}

// DefaultConfigFile is the name of the default config file.
const DefaultConfigFile = ".prettierrc"

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
