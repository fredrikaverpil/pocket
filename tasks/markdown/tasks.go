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
// Use pocket.AutoDetect(markdown.Tasks()) to enable path filtering.
func Tasks() pocket.Runnable {
	return pocket.NewTaskGroup(FormatTask()).
		DetectBy(func() []string { return []string{"."} })
}

// FormatTask returns a task that formats Markdown files using mdformat.
func FormatTask() *pocket.Task {
	return pocket.NewTask("md-format", "format Markdown files", formatAction)
}

// formatAction is the action for the md-format task.
func formatAction(ctx context.Context, tc *pocket.TaskContext) error {
	return tc.ForEachPath(ctx, func(dir string) error {
		absDir := pocket.FromGitRoot(dir)

		needsFormat, checkOutput, err := formatCheck(ctx, absDir)
		if err != nil {
			return err
		}
		if !needsFormat {
			tc.Out.Println("No files in need of formatting.")
			return nil
		}

		// Show files that need formatting in verbose mode.
		if tc.Verbose && len(checkOutput) > 0 {
			tc.Out.Printf("%s", checkOutput)
		}

		// Now actually format.
		if err := mdformat.Run(ctx, absDir); err != nil {
			return fmt.Errorf("mdformat failed in %s: %w", dir, err)
		}
		tc.Out.Println("Formatted files.")
		return nil
	})
}

// formatCheck runs mdformat --check to see if formatting is needed.
// Returns true if files need formatting, along with the check output.
func formatCheck(ctx context.Context, dir string) (needsFormat bool, output []byte, err error) {
	cmd, err := mdformat.Command(ctx, "--check", dir)
	if err != nil {
		return false, nil, fmt.Errorf("prepare mdformat: %w", err)
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	output, checkErr := cmd.CombinedOutput()
	return checkErr != nil, output, nil
}
