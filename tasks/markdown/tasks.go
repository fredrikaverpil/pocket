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
func Tasks() pocket.Runnable {
	return pocket.Serial(
		FormatTask(),
	)
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
