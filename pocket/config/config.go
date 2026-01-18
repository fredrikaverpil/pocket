package config

import (
	"context"
	"fmt"
	"os"

	"github.com/fredrikaverpil/pocket/pocket/pk"
)

// Config represents the Pocket configuration.
// It holds the root of the task graph to be executed.
type Config struct {
	// Root is the top-level runnable that composes all tasks.
	// This is typically created using pk.Serial() to compose multiple tasks.
	Root pk.Runnable
}

// Run executes the configuration's task graph.
// This is a temporary simple executor - will be replaced with
// a proper plan calculator and executor in later phases.
func (c *Config) Run(ctx context.Context) error {
	return pk.Execute(ctx, c.Root)
}

// RunMain is the main entry point for executing a Pocket configuration.
// It's called from .pocket/main.go and handles the complete execution lifecycle.
// This is a temporary implementation - will be replaced with proper CLI handling.
func RunMain(cfg *Config) {
	ctx := context.Background()

	// Execute the configuration
	if err := cfg.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
