// Package markdown provides Markdown formatting tasks.
package markdown

import (
	"github.com/fredrikaverpil/pocket/pk"
)

// Detect returns a DetectFunc for Markdown projects.
// Returns repository root since markdown files are typically scattered.
func Detect() pk.DetectFunc {
	return func(_ []string, _ string) []string {
		return []string{"."}
	}
}

// Tasks returns all markdown tasks composed as a Runnable.
//
// Use with pk.WithDetect to specify where tasks should run:
//
//	pk.WithOptions(
//	    markdown.Tasks(),
//	    pk.WithDetect(markdown.Detect()),
//	)
func Tasks() pk.Runnable {
	return Format
}
