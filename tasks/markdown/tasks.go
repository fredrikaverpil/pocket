// Package markdown provides Markdown-related build tasks.
package markdown

import (
	"context"
	"fmt"
	"slices"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/mdformat"
)

// Options defines options for a Markdown module within a task group.
type Options struct {
	// Skip lists full task names to skip (e.g., "md-format").
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
	// placeholder for future options
}

// Group defines the Markdown task group.
var Group = pocket.TaskGroupDef[Options]{
	Name:   "markdown",
	Detect: func() []string { return []string{"."} }, // Just use root for markdown.
	Tasks: []pocket.TaskDef[Options]{
		{Name: "md-format", Create: FormatTask},
	},
}

// Auto creates a Markdown task group that runs from the repository root.
// Since markdown files are typically scattered throughout a project,
// this defaults to running mdformat from root rather than detecting individual directories.
// The defaults parameter specifies default options for all detected modules.
// Skip patterns can be passed to exclude paths or specific tasks.
func Auto(defaults Options, opts ...pocket.SkipOption) pocket.TaskGroup {
	return Group.Auto(defaults, opts...)
}

// New creates a Markdown task group with explicit module configuration.
func New(modules map[string]Options) pocket.TaskGroup {
	return Group.New(modules)
}

// FormatTask returns a task that formats Markdown files using mdformat.
// The modules map specifies which directories to format and their options.
func FormatTask(modules map[string]Options) *pocket.Task {
	return &pocket.Task{
		Name:  "md-format",
		Usage: "format Markdown files",
		Action: func(ctx context.Context, _ map[string]string) error {
			for mod := range modules {
				if err := mdformat.Run(ctx, mod); err != nil {
					return fmt.Errorf("mdformat format failed in %s: %w", mod, err)
				}
			}
			return nil
		},
	}
}
