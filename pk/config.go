package pk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fredrikaverpil/pocket/internal/shim"
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

	// Generate shims (after planning, before execution)
	gitRoot := findGitRoot()
	pocketDir := filepath.Join(gitRoot, ".pocket")
	_, err = shim.GenerateShims(
		ctx,
		gitRoot,
		pocketDir,
		p.ModuleDirectories,
		shim.Config{
			Posix:      true,
			Windows:    true,
			PowerShell: true,
		},
	)
	if err != nil {
		return fmt.Errorf("generating shims: %w", err)
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
