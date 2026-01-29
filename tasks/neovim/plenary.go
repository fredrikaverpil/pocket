// Package neovim provides Neovim-related build tasks.
package neovim

import (
	"context"
	"flag"
	"fmt"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/neovim"
	"github.com/fredrikaverpil/pocket/tools/treesitter"
)

var (
	plenaryFlags       = flag.NewFlagSet("nvim-test", flag.ContinueOnError)
	plenaryBootstrap   = plenaryFlags.String("bootstrap", "spec/bootstrap.lua", "bootstrap.lua file path")
	plenaryMinimalInit = plenaryFlags.String("minimal-init", "spec/minimal_init.lua", "minimal_init.lua file path")
	plenaryTestDir     = plenaryFlags.String("test-dir", "spec/", "test directory")
	plenaryTimeout     = plenaryFlags.Int("timeout", 500000, "test timeout in ms")
)

// Context keys for configuration
type versionKey struct{}

// versionFromContext returns the Neovim version from context.
// Returns neovim.DefaultVersion if not set.
func versionFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(versionKey{}).(string); ok {
		return v
	}
	return neovim.DefaultVersion
}

// WithNeovimStable configures PlenaryTest to use stable Neovim.
// This sets version, task name suffix, and path isolation.
func WithNeovimStable() pk.PathOption {
	return pk.CombineOptions(
		pk.WithContextValue(versionKey{}, neovim.Stable),
		pk.WithName("stable"),
		pk.WithExplicitPath(".tests/stable"),
		pk.WithCleanPath(),
	)
}

// WithNeovimNightly configures PlenaryTest to use nightly Neovim.
// This sets version, task name suffix, and path isolation.
func WithNeovimNightly() pk.PathOption {
	return pk.CombineOptions(
		pk.WithContextValue(versionKey{}, neovim.Nightly),
		pk.WithName("nightly"),
		pk.WithExplicitPath(".tests/nightly"),
		pk.WithCleanPath(),
	)
}


// PlenaryTest runs Neovim plenary tests.
// Use with PathOptions to configure version:
//
//	pk.Parallel(
//	    pk.WithOptions(neovim.PlenaryTest, neovim.WithNeovimStable()),
//	    pk.WithOptions(neovim.PlenaryTest, neovim.WithNeovimNightly()),
//	)
//
// Dependencies (neovim, treesitter) are installed in parallel.
// Since they're Global tasks, each only runs once regardless of how many
// times PlenaryTest is invoked with different versions.
//
// For additional dependencies (e.g., gotestsum), compose them in your config:
//
//	pk.Serial(
//	    gotestsum.Install,
//	    pk.Parallel(
//	        pk.WithOptions(neovim.PlenaryTest, neovim.WithNeovimStable()),
//	        pk.WithOptions(neovim.PlenaryTest, neovim.WithNeovimNightly()),
//	    ),
//	)
var PlenaryTest = pk.NewTask("nvim-test", "run neovim plenary tests", plenaryFlags,
	pk.Serial(
		pk.Parallel(
			neovim.Install(neovim.Stable),
			neovim.Install(neovim.Nightly),
			treesitter.Install,
		),
		runPlenaryTests(),
	),
)

func runPlenaryTests() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		version := versionFromContext(ctx)

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
			pk.Printf(ctx, "  cwd:         %s\n", pk.FromGitRoot(pk.PathFromContext(ctx)))
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
