// Package markdown provides Markdown formatting tasks.
// This is a "task" package - it orchestrates tools to do work.
package markdown

import (
	"github.com/fredrikaverpil/pocket"
)

// Tasks returns all markdown tasks composed as a Runnable.
// Use this with pocket.Paths().DetectBy() for auto-detection.
//
// Example:
//
//	pocket.Paths(markdown.Tasks()).DetectBy(markdown.Detect())
func Tasks() pocket.Runnable {
	return Format
}

// Detect returns a detection function for Markdown projects.
// Returns repository root since markdown files are typically scattered.
func Detect() func() []string {
	return func() []string {
		return []string{"."}
	}
}
