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
// Use pocket.P(lua.Tasks()).Detect() to enable path filtering.
func Tasks() pocket.Runnable {
	return &luaTasks{
		format: FormatTask(),
	}
}

// luaTasks is the Runnable for Lua tasks that also implements Detectable.
type luaTasks struct {
	format *pocket.Task
}

// Run executes all Lua tasks.
func (l *luaTasks) Run(ctx context.Context) error {
	return l.format.Run(ctx)
}

// Tasks returns all Lua tasks.
func (l *luaTasks) Tasks() []*pocket.Task {
	return []*pocket.Task{l.format}
}

// DefaultDetect returns a function that detects Lua directories.
// Returns root since Lua files are typically scattered.
func (l *luaTasks) DefaultDetect() func() []string {
	return func() []string { return []string{"."} }
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
