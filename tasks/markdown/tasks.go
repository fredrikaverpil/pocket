// Package markdown provides Markdown-related build tasks.
package markdown

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/mdformat"
)

// Tasks returns a Runnable that executes all Markdown tasks.
// Runs from repository root since markdown files are typically scattered.
// Use pocket.P(markdown.Tasks()).Detect() to enable path filtering.
func Tasks() pocket.Runnable {
	return &mdTasks{
		format: FormatTask(),
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
func FormatTask() *pocket.Task {
	return &pocket.Task{
		Name:  "md-format",
		Usage: "format Markdown files",
		Action: func(ctx context.Context, _ map[string]string) error {
			if err := mdformat.Run(ctx, "."); err != nil {
				return fmt.Errorf("mdformat failed: %w", err)
			}
			return nil
		},
	}
}
