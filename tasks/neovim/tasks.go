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

// Re-export install tasks for convenience.
// These are Global tasks that only run once regardless of path.
var (
	InstallStable  = neovim.InstallStable
	InstallNightly = neovim.InstallNightly
)

// Detect returns a DetectFunc for Neovim plugin projects.
// Returns repository root since Lua files are typically scattered.
func Detect() pk.DetectFunc {
	return func(_ []string, _ string) []string {
		return []string{"."}
	}
}

// Tasks returns the plenary test task.
// Use WithNeovimStable() or WithNeovimNightly() to configure the version.
func Tasks() pk.Runnable {
	return PlenaryTest
}
