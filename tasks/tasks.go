// Package tasks provides the unified task entry point for bld.
// It automatically creates tasks based on the provided Config.
package tasks

import (
	"github.com/fredrikaverpil/bld"
	"github.com/fredrikaverpil/bld/tasks/generate"
	"github.com/fredrikaverpil/bld/tasks/golang"
	"github.com/fredrikaverpil/bld/tasks/lua"
	"github.com/fredrikaverpil/bld/tasks/markdown"
	"github.com/fredrikaverpil/bld/tasks/update"
	"github.com/goyek/goyek/v3"
)

// Tasks holds all registered tasks based on the Config.
type Tasks struct {
	// All runs all configured tasks.
	All *goyek.DefinedTask

	// Go holds Go-specific tasks (nil if Config.Go is nil).
	Go *golang.Tasks

	// Lua holds Lua-specific tasks (nil if Config.Lua is nil).
	Lua *lua.Tasks

	// Markdown holds Markdown-specific tasks (nil if Config.Markdown is nil).
	Markdown *markdown.Tasks

	// Generate regenerates all generated files.
	Generate *goyek.DefinedTask

	// Update updates bld and regenerates files.
	Update *goyek.DefinedTask

	// Custom holds custom tasks registered for this context.
	Custom []*goyek.DefinedTask
}

// New creates tasks based on the provided Config.
// Tasks are only created for configured languages/features.
// The config should already be filtered for the current context via Config.ForContext().
func New(cfg bld.Config) *Tasks {
	cfg = cfg.WithDefaults()
	t := &Tasks{}

	// Generate runs first - other tasks may need generated files.
	t.Generate = generate.Task(cfg)

	// Update is standalone (not part of "all")
	t.Update = update.Task(cfg)

	// Start with generate as first dep (runs before everything else)
	deps := goyek.Deps{t.Generate}

	// Create Go tasks if configured
	if cfg.Go != nil {
		t.Go = golang.NewTasks(cfg)
		deps = append(deps, t.Go.All)
	}

	// Create Lua tasks if configured
	if cfg.Lua != nil {
		t.Lua = lua.NewTasks(cfg)
		// Note: Not added to "all" yet, use "lua-fmt" explicitly
	}

	// Create Markdown tasks if configured
	if cfg.Markdown != nil {
		t.Markdown = markdown.NewTasks(cfg)
		deps = append(deps, t.Markdown.All)
	}

	// Future: Python, etc.
	// if cfg.Python != nil {
	//     t.Python = python.NewTasks(cfg)
	//     deps = append(deps, t.Python.All)
	// }

	// Define custom tasks from config and add them to deps.
	for _, task := range cfg.CustomTasks() {
		defined := goyek.Define(task)
		t.Custom = append(t.Custom, defined)
		deps = append(deps, defined)
	}

	// Create the "all" task that runs everything.
	t.All = goyek.Define(goyek.Task{
		Name:  "all",
		Usage: "run all tasks",
		Deps:  deps,
	})

	return t
}
