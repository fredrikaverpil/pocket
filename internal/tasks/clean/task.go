// Package clean provides a task that removes downloaded tools and binaries.
package clean

import (
	"fmt"
	"os"

	"github.com/fredrikaverpil/pocket"
)

// Task returns a task that removes .pocket/tools and .pocket/bin directories.
func Task() *pocket.Task {
	return pocket.NewTask("clean", "remove .pocket/tools and .pocket/bin directories", cleanAction).
		AsBuiltin()
}

func cleanAction(rc *pocket.RunContext) error {
	ctx := rc.Context()

	// Remove .pocket/tools
	toolsDir := pocket.FromToolsDir()
	if _, err := os.Stat(toolsDir); err == nil {
		if err := os.RemoveAll(toolsDir); err != nil {
			return fmt.Errorf("remove tools dir: %w", err)
		}
		pocket.Printf(ctx, "Removed %s\n", toolsDir)
	}

	// Remove .pocket/bin
	binDir := pocket.FromBinDir()
	if _, err := os.Stat(binDir); err == nil {
		if err := os.RemoveAll(binDir); err != nil {
			return fmt.Errorf("remove bin dir: %w", err)
		}
		pocket.Printf(ctx, "Removed %s\n", binDir)
	}

	return nil
}
