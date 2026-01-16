// Package lua provides Lua-related build tasks.
package lua

import (
	"github.com/fredrikaverpil/pocket"
)

// Option configures the lua task group.
type Option func(*config)

type config struct {
	format FormatOptions
}

// WithFormat sets options for the lua-format task.
func WithFormat(opts FormatOptions) Option {
	return func(c *config) { c.format = opts }
}

// Tasks returns a Runnable that executes all Lua tasks.
// Runs from repository root since Lua files are typically scattered.
// Use pocket.Paths(lua.Tasks()).DetectBy(lua.Detect()) to enable path filtering.
//
// Execution order: format runs first, then lint.
//
// Example with options:
//
//	pocket.Paths(lua.Tasks(
//	    lua.WithFormat(lua.FormatOptions{StyluaConfig: ".stylua.toml"}),
//	)).DetectBy(lua.Detect())
func Tasks(opts ...Option) pocket.Runnable {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	formatTask := Format
	if cfg.format != (FormatOptions{}) {
		formatTask = Format.With(cfg.format)
	}

	return pocket.Serial(formatTask)
}

// Detect returns a detection function that finds Lua projects.
// It returns the repository root since Lua files are typically scattered.
//
// Usage:
//
//	pocket.Paths(lua.Tasks()).DetectBy(lua.Detect())
func Detect() func() []string {
	return func() []string {
		return []string{"."}
	}
}
