// Package claude provides tasks for Claude skill validation.
package claude

import "github.com/fredrikaverpil/pocket/pk"

// Tasks returns Claude-related tasks composed together.
func Tasks() pk.Runnable {
	return SkillValidator
}
