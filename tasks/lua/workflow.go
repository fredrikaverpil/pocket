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

// Workflow returns a Runnable that executes all Lua tasks.
// Runs from repository root since Lua files are typically scattered.
// Use pocket.Paths(lua.Workflow()).DetectBy(lua.Detect()) to enable path filtering.
//
// Example with options:
//
//	pocket.Paths(lua.Workflow(
//	    lua.WithFormat(lua.FormatOptions{StyluaConfig: ".stylua.toml"}),
//	)).DetectBy(lua.Detect())
func Workflow(opts ...Option) pocket.Runnable {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	formatTask := Format
	if cfg.format != (FormatOptions{}) {
		formatTask = Format.With(cfg.format)
	}

	return formatTask
}

// Detect returns a detection function that finds Lua projects.
// It returns the repository root since Lua files are typically scattered.
//
// Usage:
//
//	pocket.Paths(lua.Workflow()).DetectBy(lua.Detect())
func Detect() func() []string {
	return func() []string {
		return []string{"."}
	}
}
