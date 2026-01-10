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

// FormatTask returns a task that formats Markdown files using mdformat.
func FormatTask() *pocket.Task {
	return &pocket.Task{
		Name:  "md-format",
		Usage: "format Markdown files",
		Action: func(ctx context.Context, rc *pocket.RunContext) error {
			return rc.ForEachPath(func(dir string) error {
				absDir := pocket.FromGitRoot(dir)

				needsFormat, checkOutput, err := formatCheck(ctx, absDir)
				if err != nil {
					return err
				}
				if !needsFormat {
					pocket.Println(ctx, "No files in need of formatting.")
					return nil
				}

				// Show files that need formatting in verbose mode.
				if rc.Verbose && len(checkOutput) > 0 {
					pocket.Printf(ctx, "%s", checkOutput)
				}

				// Now actually format.
				if err := mdformat.Run(ctx, absDir); err != nil {
					return fmt.Errorf("mdformat failed in %s: %w", dir, err)
				}
				pocket.Println(ctx, "Formatted files.")
				return nil
			})
		},
	}
}
