// Package lua provides Lua-related build tasks.
package lua

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/stylua"
)

// Tasks returns a Runnable that executes all Lua tasks.
// Runs from repository root since Lua files are typically scattered.
func Tasks() pocket.Runnable {
	return pocket.Serial(
		FormatTask(),
	)
}

// FormatTask returns a task that formats Lua files using stylua.
func FormatTask() *pocket.Task {
	return &pocket.Task{
		Name:  "lua-format",
		Usage: "format Lua files",
		Action: func(ctx context.Context, _ map[string]string) error {
			configPath, err := stylua.ConfigPath()
			if err != nil {
				return fmt.Errorf("get stylua config: %w", err)
			}
			if err := stylua.Run(ctx, "-f", configPath, "."); err != nil {
				return fmt.Errorf("stylua format failed: %w", err)
			}
			return nil
		},
	}
}
