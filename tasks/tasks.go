// Package tasks provides the unified task entry point for bld.
// It automatically creates tasks based on the provided Config.
package tasks

import (
	"github.com/fredrikaverpil/bld"
	"github.com/fredrikaverpil/bld/tasks/generate"
	"github.com/fredrikaverpil/bld/tasks/golang"
	"github.com/fredrikaverpil/bld/tasks/markdown"
	"github.com/goyek/goyek/v3"
)

// Tasks holds all registered tasks based on the Config.
type Tasks struct {
	// All runs all configured tasks.
	All *goyek.DefinedTask

	// Go holds Go-specific tasks (nil if Config.Go is nil).
	Go *golang.Tasks

	// Markdown holds Markdown-specific tasks (nil if Config.Markdown is nil).
	Markdown *markdown.Tasks

	// Generate regenerates all generated files.
	Generate *goyek.DefinedTask
}

// New creates tasks based on the provided Config.
// Tasks are only created for configured languages/features.
func New(cfg bld.Config) *Tasks {
	cfg = cfg.WithDefaults()
	t := &Tasks{}
	var deps goyek.Deps

	// Create Go tasks if configured
	if cfg.Go != nil {
		t.Go = golang.NewTasks(cfg)
		deps = append(deps, t.Go.All)
	}

	// Create Markdown tasks if configured
	if cfg.Markdown != nil {
		t.Markdown = markdown.NewTasks(cfg)
		deps = append(deps, t.Markdown.All)
	}

	// Future: Python, Lua, etc.
	// if cfg.Python != nil {
	//     t.Python = python.NewTasks(cfg)
	//     deps = append(deps, t.Python.All)
	// }

	// Always create generate task
	t.Generate = generate.Task(cfg)
	deps = append(deps, t.Generate)

	// Create the "all" task that runs everything
	t.All = goyek.Define(goyek.Task{
		Name:  "all",
		Usage: "run all tasks",
		Deps:  deps,
	})

	return t
}
