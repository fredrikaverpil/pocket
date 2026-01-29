// Package neovim provides Neovim-related build tasks.
package neovim

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/neovim"
)

var (
	plenaryFlags       = flag.NewFlagSet("nvim-test", flag.ContinueOnError)
	plenaryBootstrap   = plenaryFlags.String("bootstrap", "spec/bootstrap.lua", "bootstrap.lua file path")
	plenaryMinimalInit = plenaryFlags.String("minimal-init", "spec/minimal_init.lua", "minimal_init.lua file path")
	plenaryTestDir     = plenaryFlags.String("test-dir", "spec/", "test directory")
	plenaryTimeout     = plenaryFlags.Int("timeout", 500000, "test timeout in ms")
)

// Context keys for configuration.
type versionKey struct{}
type siteDirKey struct{}

// versionFromContext returns the Neovim version from context.
// Returns neovim.DefaultVersion if not set.
func versionFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(versionKey{}).(string); ok {
		return v
	}
	return neovim.DefaultVersion
}

// siteDirFromContext returns the site directory from context.
// Returns ".tests/default" if not set.
func siteDirFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(siteDirKey{}).(string); ok {
		return v
	}
	return ".tests/default"
}

// WithNeovimStable configures PlenaryTest to use stable Neovim.
// This sets version, task name suffix, and site directory for isolation.
// Tests run from the git root but plugins are installed in .tests/stable/.
func WithNeovimStable() pk.PathOption {
	return pk.CombineOptions(
		pk.WithContextValue(versionKey{}, neovim.Stable),
		pk.WithContextValue(siteDirKey{}, ".tests/stable"),
		pk.WithName("stable"),
	)
}

// WithNeovimNightly configures PlenaryTest to use nightly Neovim.
// This sets version, task name suffix, and site directory for isolation.
// Tests run from the git root but plugins are installed in .tests/nightly/.
func WithNeovimNightly() pk.PathOption {
	return pk.CombineOptions(
		pk.WithContextValue(versionKey{}, neovim.Nightly),
		pk.WithContextValue(siteDirKey{}, ".tests/nightly"),
		pk.WithName("nightly"),
	)
}

// PlenaryTest runs Neovim plenary tests.
// Dependencies (neovim, treesitter, gotestsum) should be composed explicitly:
//
//	pk.Serial(
//	    gotestsum.Install,
//	    pk.Parallel(
//	        neovim.InstallStable,
//	        neovim.InstallNightly,
//	        treesitter.Install,
//	    ),
//	    pk.Parallel(
//	        pk.WithOptions(neovim.PlenaryTest, neovim.WithNeovimStable()),
//	        pk.WithOptions(neovim.PlenaryTest, neovim.WithNeovimNightly()),
//	    ),
//	)
//
// This explicit composition ensures:
//   - Install tasks are outside WithOptions, avoiding suffix propagation
//   - Each installer has a unique name (install:nvim-stable, install:nvim-nightly)
//   - Global deduplication works naturally
//   - Serial ensures installs complete before tests start
var PlenaryTest = pk.NewTask("nvim-test", "run neovim plenary tests", plenaryFlags,
	runPlenaryTests(),
)

func runPlenaryTests() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := versionFromContext(ctx)
		siteDir := siteDirFromContext(ctx)

		// Clean and create site directory for isolation.
		absSiteDir := pk.FromGitRoot(siteDir)
		if err := os.RemoveAll(absSiteDir); err != nil {
			return fmt.Errorf("clean site directory: %w", err)
		}
		if err := os.MkdirAll(absSiteDir, 0o755); err != nil {
			return fmt.Errorf("create site directory: %w", err)
		}

		// Set NEOTEST_SITE_DIR so bootstrap.lua uses our isolated directory.
		ctx = pk.WithEnv(ctx, fmt.Sprintf("NEOTEST_SITE_DIR=%s", absSiteDir))

		// Resolve paths from git root so they work regardless of execution directory.
		bootstrap := pk.FromGitRoot(*plenaryBootstrap)
		minimalInit := pk.FromGitRoot(*plenaryMinimalInit)
		testDir := pk.FromGitRoot(*plenaryTestDir)

		// Use the specific neovim binary for this version to avoid symlink collisions
		// when running multiple versions in parallel.
		nvimBinary := neovim.BinaryPath(version)

		if pk.Verbose(ctx) {
			pk.Printf(ctx, "  nvim:        %s\n", nvimBinary)
			pk.Printf(ctx, "  bootstrap:   %s\n", bootstrap)
			pk.Printf(ctx, "  minimal_init: %s\n", minimalInit)
			pk.Printf(ctx, "  test_dir:    %s\n", testDir)
			pk.Printf(ctx, "  timeout:     %d\n", *plenaryTimeout)
			pk.Printf(ctx, "  site_dir:    %s\n", absSiteDir)
		}

		plenaryCmd := fmt.Sprintf(
			"PlenaryBustedDirectory %s { minimal_init = '%s', timeout = %d }",
			testDir, minimalInit, *plenaryTimeout,
		)

		return pk.Exec(ctx, nvimBinary,
			"--headless",
			"--noplugin",
			"-i", "NONE",
			"-u", bootstrap,
			"-c", plenaryCmd,
		)
	})
}
