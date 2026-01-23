package neovim

import (
	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/neovim"
)

// Re-export version constants for convenience.
const (
	Stable  = neovim.Stable
	Nightly = neovim.Nightly
)

// Detect returns a DetectFunc for Neovim plugin projects.
// Returns repository root since Lua files are typically scattered.
func Detect() pk.DetectFunc {
	return func(_ []string, _ string) []string {
		return []string{"."}
	}
}

// Tasks returns plenary test task with default configuration.
// For custom configuration, use Test() directly with options.
func Tasks() pk.Runnable {
	return Test()
}
