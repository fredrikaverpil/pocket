// Package neovim provides Neovim-related build tasks.
package neovim

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/neovim"
	"github.com/fredrikaverpil/pocket/tools/treesitter"
)

// PlenaryTestConfig configures plenary test execution.
type PlenaryTestConfig struct {
	// Bootstrap is the path to the bootstrap.lua file (runs once before all tests).
	// This file typically sets up the test environment, downloads plugins, etc.
	Bootstrap string

	// MinimalInit is the path to the minimal_init.lua file (runs before each test).
	// This file configures the minimal Neovim environment for each test.
	MinimalInit string

	// TestDir is the directory containing test files (default: "spec/").
	TestDir string

	// Timeout is the test timeout in milliseconds (default: 500000).
	Timeout int

	// Version is the Neovim version to use (default: neovim.DefaultVersion).
	// Supports: "stable", "nightly", or specific versions like "v0.10.0".
	Version string
}

// PlenaryTestOpt configures plenary test execution.
type PlenaryTestOpt func(*PlenaryTestConfig)

// WithPlenaryBootstrap sets the bootstrap.lua file path.
func WithPlenaryBootstrap(path string) PlenaryTestOpt {
	return func(cfg *PlenaryTestConfig) {
		cfg.Bootstrap = path
	}
}

// WithPlenaryMinimalInit sets the minimal_init.lua file path.
func WithPlenaryMinimalInit(path string) PlenaryTestOpt {
	return func(cfg *PlenaryTestConfig) {
		cfg.MinimalInit = path
	}
}

// WithPlenaryTestDir sets the test directory.
func WithPlenaryTestDir(dir string) PlenaryTestOpt {
	return func(cfg *PlenaryTestConfig) {
		cfg.TestDir = dir
	}
}

// WithPlenaryTimeout sets the test timeout in milliseconds.
func WithPlenaryTimeout(ms int) PlenaryTestOpt {
	return func(cfg *PlenaryTestConfig) {
		cfg.Timeout = ms
	}
}

// WithPlenaryNvimVersion sets the Neovim version.
func WithPlenaryNvimVersion(version string) PlenaryTestOpt {
	return func(cfg *PlenaryTestConfig) {
		cfg.Version = version
	}
}

// WithPlenaryTestOption returns a PathOption that isolates plenary test execution.
// Each version gets its own directory (.tests/{version}/) which is cleaned
// before running. This allows stable and nightly to run in parallel.
func WithPlenaryTestOption(version string) pk.PathOption {
	return pk.CombineOptions(
		pk.WithExplicitPath(fmt.Sprintf(".tests/%s", version)),
		pk.WithCleanPath(),
	)
}

func newPlenaryTestConfig(opts []PlenaryTestOpt) *PlenaryTestConfig {
	cfg := &PlenaryTestConfig{
		Bootstrap:   "spec/bootstrap.lua",
		MinimalInit: "spec/minimal_init.lua",
		TestDir:     "spec/",
		Timeout:     500000,
		Version:     neovim.DefaultVersion,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// PlenaryTest creates a task that runs Neovim plenary tests.
//
// Example usage:
//
//	neovim.PlenaryTest() // uses defaults: spec/bootstrap.lua, spec/minimal_init.lua, spec/
//
//	neovim.PlenaryTest(
//	    neovim.WithBootstrap("test/bootstrap.lua"),
//	    neovim.WithMinimalInit("test/minimal_init.lua"),
//	    neovim.WithTestDir("test/"),
//	    neovim.WithVersion(neovim.Nightly),
//	)
func PlenaryTest(opts ...PlenaryTestOpt) *pk.Task {
	cfg := newPlenaryTestConfig(opts)

	taskName := "nvim-test"
	if cfg.Version != neovim.DefaultVersion {
		taskName = fmt.Sprintf("nvim-test-%s", cfg.Version)
	}

	return pk.NewTask(taskName, "run neovim plenary tests", nil,
		pk.Serial(
			pk.Parallel(
				neovim.Install(cfg.Version),
				treesitter.Install, // Required for nvim-treesitter parser compilation
			),
			runPlenaryTests(cfg),
		),
	)
}

func runPlenaryTests(cfg *PlenaryTestConfig) pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		// Resolve paths from git root so they work regardless of execution directory.
		bootstrap := pk.FromGitRoot(cfg.Bootstrap)
		minimalInit := pk.FromGitRoot(cfg.MinimalInit)
		testDir := pk.FromGitRoot(cfg.TestDir)

		// Build the PlenaryBustedDirectory command
		// nvim --headless --noplugin -i NONE -u {bootstrap} \
		//   -c "PlenaryBustedDirectory {dir} { minimal_init = '{init}', timeout = {timeout} }"
		plenaryCmd := fmt.Sprintf(
			"PlenaryBustedDirectory %s { minimal_init = '%s', timeout = %d }",
			testDir, minimalInit, cfg.Timeout,
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
