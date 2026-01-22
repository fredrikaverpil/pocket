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
func Tasks() pk.Runnable {
	return pk.WithOptions(
		Format,
		pk.WithDetect(Detect()),
	)
}
