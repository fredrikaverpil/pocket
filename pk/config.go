package pk

import (
	"context"
	"fmt"
	"os"
)

// Config represents the Pocket configuration.
// It holds the root of the task graph to be executed.
type Config struct {
	// Root is the top-level runnable that composes all tasks.
	// This is typically created using Serial() to compose multiple tasks.
	Root Runnable
}

func (c *Config) Execute(ctx context.Context) error {
	if c.Root == nil {
		return nil
	}

	// Build plan once
	p, err := NewPlan(c.Root)
	if err != nil {
		return err
	}

	// Execute with plan in context
	ctx = WithPlan(ctx, p)
	return c.Root.run(ctx)
}

// RunMain is the main entry point for executing a Pocket configuration.
// It's called from .pocket/main.go and handles the complete execution lifecycle.
// This is a temporary implementation - will be replaced with proper CLI handling.
func RunMain(cfg *Config) {
	ctx := context.Background()
	if err := cfg.Execute(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
