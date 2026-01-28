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

// WithPlenaryVersion returns a PathOption that isolates plenary test execution.
// Each version gets its own directory (.tests/{version}/) which is cleaned
// before running. This allows stable and nightly to run in parallel.
func WithPlenaryVersion(version string) pk.PathOption {
	return pk.CombineOptions(
		pk.WithExplicitPath(fmt.Sprintf(".tests/%s", version)),
		pk.WithCleanPath(),
	)
}

// PlenaryTest creates a task that runs Neovim plenary tests.
// Version determines which Neovim binary to install.
//
// Use pk.WithOptions with WithPlenaryVersion to isolate parallel execution:
//
//	pk.Parallel(
//	    pk.WithOptions(
//	        neovim.PlenaryTest(neovim.Stable),
//	        neovim.WithPlenaryVersion(neovim.Stable),
//	    ),
//	    pk.WithOptions(
//	        neovim.PlenaryTest(neovim.Nightly),
//	        neovim.WithPlenaryVersion(neovim.Nightly),
//	    ),
//	)
func PlenaryTest(version string) *pk.Task {
	if version == "" {
		version = neovim.DefaultVersion
	}

	taskName := "nvim-test"
	if version != neovim.DefaultVersion {
		taskName = fmt.Sprintf("nvim-test-%s", version)
	}

	return pk.NewTask(taskName, "run neovim plenary tests", plenaryFlags,
		pk.Serial(
			pk.Parallel(
				neovim.Install(version),
				treesitter.Install, // Required for nvim-treesitter parser compilation
			),
			runPlenaryTests(),
		),
	)
}

func runPlenaryTests() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		// Resolve paths from git root so they work regardless of execution directory.
		bootstrap := pk.FromGitRoot(*plenaryBootstrap)
		minimalInit := pk.FromGitRoot(*plenaryMinimalInit)
		testDir := pk.FromGitRoot(*plenaryTestDir)

		plenaryCmd := fmt.Sprintf(
			"PlenaryBustedDirectory %s { minimal_init = '%s', timeout = %d }",
			testDir, minimalInit, *plenaryTimeout,
		)

		return pk.Exec(ctx, neovim.Name,
			"--headless",
			"--noplugin",
			"-i", "NONE",
			"-u", bootstrap,
			"-c", plenaryCmd,
		)
	})
}
