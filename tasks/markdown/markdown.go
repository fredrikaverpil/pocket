// Package markdown provides Markdown formatting tasks.
// This is a "task" package - it orchestrates tools to do work.
package markdown

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/prettier"
)

// Format formats Markdown files using prettier.
var Format = pocket.Func("md-format", "format Markdown files", format)

// FormatOptions configures markdown formatting.
type FormatOptions struct {
	Check bool // check only, don't write
}

// Workflow returns all markdown tasks composed as a Runnable.
// Use this with pocket.Paths().DetectBy() for auto-detection.
//
// Example:
//
//	pocket.Paths(markdown.Workflow()).DetectBy(markdown.Detect())
func Workflow() pocket.Runnable {
	return Format
}

// Detect returns a detection function for Markdown projects.
// Returns repository root since markdown files are typically scattered.
func Detect() func() []string {
	return func() []string {
		return []string{"."}
	}
}

func format(ctx context.Context) error {
	opts := pocket.Options[FormatOptions](ctx)

	args := []string{}
	if opts.Check {
		args = append(args, "--check")
	} else {
		args = append(args, "--write")
	}

	// Add config if available
	if configPath, err := pocket.ConfigPath("prettier", prettier.Config); err == nil && configPath != "" {
		args = append(args, "--config", configPath)
	}

	// Add ignore file if available
	if ignorePath, err := prettier.EnsureIgnoreFile(); err == nil {
		args = append(args, "--ignore-path", ignorePath)
	}

	args = append(args, "**/*.md")

	if err := prettier.Exec(ctx, args...); err != nil {
		return fmt.Errorf("prettier failed: %w", err)
	}
	return nil
}
