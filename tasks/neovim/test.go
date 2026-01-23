// Package neovim provides Neovim-related build tasks.
package neovim

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/neovim"
	"github.com/fredrikaverpil/pocket/tools/treesitter"
)

// TestConfig configures plenary test execution.
type TestConfig struct {
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

// TestOpt configures plenary test execution.
type TestOpt func(*TestConfig)

// WithBootstrap sets the bootstrap.lua file path.
func WithBootstrap(path string) TestOpt {
	return func(cfg *TestConfig) {
		cfg.Bootstrap = path
	}
}

// WithMinimalInit sets the minimal_init.lua file path.
func WithMinimalInit(path string) TestOpt {
	return func(cfg *TestConfig) {
		cfg.MinimalInit = path
	}
}

// WithTestDir sets the test directory.
func WithTestDir(dir string) TestOpt {
	return func(cfg *TestConfig) {
		cfg.TestDir = dir
	}
}

// WithTimeout sets the test timeout in milliseconds.
func WithTimeout(ms int) TestOpt {
	return func(cfg *TestConfig) {
		cfg.Timeout = ms
	}
}

// WithVersion sets the Neovim version.
func WithVersion(version string) TestOpt {
	return func(cfg *TestConfig) {
		cfg.Version = version
	}
}

func newTestConfig(opts []TestOpt) *TestConfig {
	cfg := &TestConfig{
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

// Test creates a task that runs Neovim plenary tests.
//
// Example usage:
//
//	neovim.Test() // uses defaults: spec/bootstrap.lua, spec/minimal_init.lua, spec/
//
//	neovim.Test(
//	    neovim.WithBootstrap("test/bootstrap.lua"),
//	    neovim.WithMinimalInit("test/minimal_init.lua"),
//	    neovim.WithTestDir("test/"),
//	    neovim.WithVersion(neovim.Nightly),
//	)
func Test(opts ...TestOpt) *pk.Task {
	cfg := newTestConfig(opts)

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

func runPlenaryTests(cfg *TestConfig) pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		// Build the PlenaryBustedDirectory command
		// nvim --headless --noplugin -i NONE -u {bootstrap} \
		//   -c "PlenaryBustedDirectory {dir} { minimal_init = '{init}', timeout = {timeout} }"
		plenaryCmd := fmt.Sprintf(
			"PlenaryBustedDirectory %s { minimal_init = '%s', timeout = %d }",
			cfg.TestDir, cfg.MinimalInit, cfg.Timeout,
		)

		return pk.Exec(ctx, neovim.Name,
			"--headless",
			"--noplugin",
			"-i", "NONE",
			"-u", cfg.Bootstrap,
			"-c", plenaryCmd,
		)
	})
}
