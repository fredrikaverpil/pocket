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
// Use pocket.Paths(markdown.Tasks()).DetectBy(markdown.Detect()) to enable path filtering.
func Tasks() pocket.Runnable {
	return FormatTask()
}

// Detect returns a detection function that finds Markdown projects.
// It returns the repository root since markdown files are typically scattered.
//
// Usage:
//
//	pocket.Paths(markdown.Tasks()).DetectBy(markdown.Detect())
func Detect() func() []string {
	return func() []string {
		return []string{"."}
	}
}

// FormatTask returns a task that formats Markdown files using mdformat.
func FormatTask() *pocket.Task {
	return pocket.NewTask("md-format", "format Markdown files", formatAction)
}

// formatAction is the action for the md-format task.
func formatAction(ctx context.Context, tc *pocket.TaskContext) error {
	absDir := pocket.FromGitRoot(tc.Path)
	if err := mdformat.Tool.Exec(ctx, tc, "--number", "--wrap", "80", absDir); err != nil {
		return fmt.Errorf("mdformat failed in %s: %w", tc.Path, err)
	}
	return nil
}
