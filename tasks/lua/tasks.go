// Package lua provides Lua-related build tasks.
package lua

import (
	"context"
	"fmt"
	"slices"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/stylua"
)

// Options defines options for a Lua module within a task group.
type Options struct {
	// Skip lists full task names to skip (e.g., "lua-format").
	Skip []string

	// Task-specific options
	Format FormatOptions
}

// ShouldRun returns true if the given task should run based on the Skip list.
func (o Options) ShouldRun(taskName string) bool {
	return !slices.Contains(o.Skip, taskName)
}

// FormatOptions defines options for the format task.
type FormatOptions struct {
	// ConfigFile overrides the default stylua config file.
	ConfigFile string
}

// Group defines the Lua task group.
var Group = pocket.TaskGroupDef[Options]{
	Name:   "lua",
	Detect: func() []string { return []string{"."} }, // Run from root.
	Tasks: []pocket.TaskDef[Options]{
		{Name: "lua-format", Create: FormatTask},
	},
}

// Auto creates a Lua task group that runs from the repository root.
// Since Lua files are typically scattered throughout a project,
// this defaults to running stylua from root rather than detecting individual directories.
// The defaults parameter specifies default options.
// Skip patterns can be passed to exclude specific tasks.
func Auto(defaults Options, opts ...pocket.SkipOption) pocket.TaskGroup {
	return Group.Auto(defaults, opts...)
}

// New creates a Lua task group with explicit module configuration.
func New(modules map[string]Options) pocket.TaskGroup {
	return Group.New(modules)
}

// FormatTask returns a task that formats Lua files using stylua.
// The modules map specifies which directories to format and their options.
func FormatTask(modules map[string]Options) *pocket.Task {
	return &pocket.Task{
		Name:  "lua-format",
		Usage: "format Lua files",
		Action: func(ctx context.Context, _ map[string]string) error {
			for mod, opts := range modules {
				configPath := opts.Format.ConfigFile
				if configPath == "" {
					var err error
					configPath, err = stylua.ConfigPath()
					if err != nil {
						return fmt.Errorf("get stylua config: %w", err)
					}
				}
				if err := stylua.Run(ctx, "-f", configPath, mod); err != nil {
					return fmt.Errorf("stylua format failed in %s: %w", mod, err)
				}
			}
			return nil
		},
	}
}
