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
//
// Use with pk.WithDetect to specify where tasks should run:
//
//	pk.WithOptions(
//	    lua.Tasks(),
//	    pk.WithDetect(lua.Detect()),
//	)
func Tasks() pk.Runnable {
	return pk.Parallel(Format, QueryFormat, QueryLint)
}
