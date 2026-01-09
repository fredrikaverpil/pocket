// Package lua provides Lua-related build tasks.
package lua

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/stylua"
)

// Options configures the Lua tasks.
type Options struct{}

// Tasks returns a Runnable that executes all Lua tasks.
// Runs from repository root since Lua files are typically scattered.
// Use pocket.AutoDetect(lua.Tasks()) to enable path filtering.
func Tasks(opts ...Options) pocket.Runnable {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	return &luaTasks{
		format: FormatTask(o),
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
func FormatTask(_ Options) *pocket.Task {
	return &pocket.Task{
		Name:  "lua-format",
		Usage: "format Lua files",
		Action: func(ctx context.Context, opts *pocket.RunContext) error {
			configPath, err := stylua.ConfigPath()
			if err != nil {
				return fmt.Errorf("get stylua config: %w", err)
			}
			for _, dir := range opts.Paths {
				if err := stylua.Run(ctx, "-f", configPath, pocket.FromGitRoot(dir)); err != nil {
					return fmt.Errorf("stylua format failed in %s: %w", dir, err)
				}
			}
			return nil
		},
	}
}
