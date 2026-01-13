// Package lua provides Lua-related build tasks.
package lua

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/stylua"
)

// TasksOption configures the lua task group.
type TasksOption func(*tasksConfig)

type tasksConfig struct {
	format FormatOptions
}

// WithFormat sets options for the lua-format task.
func WithFormat(opts FormatOptions) TasksOption {
	return func(c *tasksConfig) { c.format = opts }
}

// Tasks returns a Runnable that executes all Lua tasks.
// Runs from repository root since Lua files are typically scattered.
// Use pocket.Paths(lua.Tasks()).DetectBy(lua.Detect()) to enable path filtering.
//
// Example with options:
//
//	pocket.Paths(lua.Tasks(
//	    lua.WithFormat(lua.FormatOptions{StyluaConfig: ".stylua.toml"}),
//	)).DetectBy(lua.Detect())
func Tasks(opts ...TasksOption) pocket.Runnable {
	var cfg tasksConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	return FormatTask().WithOptions(cfg.format)
}

// Detect returns a detection function that finds Lua projects.
// It returns the repository root since Lua files are typically scattered.
//
// Usage:
//
//	pocket.Paths(lua.Tasks()).DetectBy(lua.Detect())
func Detect() func() []string {
	return func() []string {
		return []string{"."}
	}
}

// FormatOptions configures the lua-format task.
type FormatOptions struct {
	StyluaConfig string `usage:"path to stylua config file"`
}

// FormatTask returns a task that formats Lua files using stylua.
// Use WithOptions to set project-level configuration.
func FormatTask() *pocket.Task {
	return pocket.NewTask("lua-format", "format Lua files", formatAction)
}

// formatAction is the action for the lua-format task.
func formatAction(ctx context.Context, tc *pocket.TaskContext) error {
	opts := pocket.GetOptions[FormatOptions](tc)
	configPath := opts.StyluaConfig
	if configPath == "" {
		var err error
		configPath, err = stylua.Tool.ConfigPath()
		if err != nil {
			return fmt.Errorf("get stylua config: %w", err)
		}
	}

	absDir := pocket.FromGitRoot(tc.Path)
	if err := stylua.Tool.Exec(ctx, tc, "-f", configPath, absDir); err != nil {
		return fmt.Errorf("stylua format failed in %s: %w", tc.Path, err)
	}
	return nil
}
