// Package lua provides Lua-related build tasks.
package lua

import (
	"github.com/fredrikaverpil/pocket/pk"
)

// Detect returns a DetectFunc for Lua projects.
// Returns repository root since Lua files are typically scattered.
func Detect() pk.DetectFunc {
	return func(_ []string, _ string) []string {
		return []string{"."}
	}
}

// Tasks returns all Lua tasks composed as a Runnable.
func Tasks() pk.Runnable {
	return pk.WithOptions(
		Format,
		pk.WithDetect(Detect()),
	)
}
