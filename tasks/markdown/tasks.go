// Package markdown provides Markdown-related build tasks.
package markdown

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/mdformat"
)

// Options configures the Markdown tasks.
type Options struct{}

// Tasks returns a Runnable that executes all Markdown tasks.
// Runs from repository root since markdown files are typically scattered.
// Use pocket.AutoDetect(markdown.Tasks()) to enable path filtering.
func Tasks(opts ...Options) pocket.Runnable {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	return &mdTasks{
		format: FormatTask(o),
	}
}

// mdTasks is the Runnable for Markdown tasks that also implements Detectable.
type mdTasks struct {
	format *pocket.Task
}

// Run executes all Markdown tasks.
func (m *mdTasks) Run(ctx context.Context) error {
	return m.format.Run(ctx)
}

// Tasks returns all Markdown tasks.
func (m *mdTasks) Tasks() []*pocket.Task {
	return []*pocket.Task{m.format}
}

// DefaultDetect returns a function that detects Markdown directories.
// Returns root since markdown files are typically scattered.
func (m *mdTasks) DefaultDetect() func() []string {
	return func() []string { return []string{"."} }
}

// FormatTask returns a task that formats Markdown files using mdformat.
func FormatTask(_ Options) *pocket.Task {
	return &pocket.Task{
		Name:  "md-format",
		Usage: "format Markdown files",
		Action: func(ctx context.Context, opts *pocket.RunContext) error {
			for _, dir := range opts.Paths {
				if err := mdformat.Run(ctx, pocket.FromGitRoot(dir)); err != nil {
					return fmt.Errorf("mdformat failed in %s: %w", dir, err)
				}
			}
			return nil
		},
	}
}
