// Package tasks provides the unified task entry point for pocket.
// It automatically creates tasks based on the provided Config.
package tasks

import (
	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tasks/generate"
	"github.com/fredrikaverpil/pocket/tasks/gitdiff"
	"github.com/fredrikaverpil/pocket/tasks/golang"
	"github.com/fredrikaverpil/pocket/tasks/lua"
	"github.com/fredrikaverpil/pocket/tasks/markdown"
	"github.com/fredrikaverpil/pocket/tasks/update"
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

	// Update updates pocket and regenerates files.
	Update *goyek.DefinedTask

	// GitDiff fails if there are uncommitted changes.
	GitDiff *goyek.DefinedTask

	// Custom holds custom tasks registered for this context.
	Custom []*goyek.DefinedTask
}

// New creates tasks based on the provided Config.
// Tasks are only created for configured languages/features.
// The config should already be filtered for the current context via Config.ForContext().
func New(cfg pocket.Config) *Tasks {
	cfg = cfg.WithDefaults()
	t := &Tasks{}

	// Generate runs first - other tasks may need generated files.
	t.Generate = generate.Task(cfg)

	// Update is standalone (not part of "all")
	t.Update = update.Task(cfg)

	// Start with generate as first dep (runs before everything else)
	deps := goyek.Deps{t.Generate}

	// Create Go tasks if configured.
	// Only add tasks to "all" deps if there are modules configured for that task.
	if cfg.Go != nil {
		t.Go = golang.NewTasks(cfg)
		if len(cfg.GoModulesForFormat()) > 0 {
			deps = append(deps, t.Go.Format)
		}
		if len(cfg.GoModulesForLint()) > 0 {
			deps = append(deps, t.Go.Lint)
		}
		if len(cfg.GoModulesForTest()) > 0 {
			deps = append(deps, t.Go.Test)
		}
		if len(cfg.GoModulesForVulncheck()) > 0 {
			deps = append(deps, t.Go.Vulncheck)
		}
	}

	// Create Lua tasks if configured.
	if cfg.Lua != nil {
		t.Lua = lua.NewTasks(cfg)
		if len(cfg.LuaModulesForFormat()) > 0 {
			deps = append(deps, t.Lua.Format)
		}
	}

	// Create Markdown tasks if configured.
	if cfg.Markdown != nil {
		t.Markdown = markdown.NewTasks(cfg)
		if len(cfg.MarkdownModulesForFormat()) > 0 {
			deps = append(deps, t.Markdown.Format)
		}
	}

	// Define custom tasks from config and add them to deps.
	for _, task := range cfg.CustomTasks() {
		defined := goyek.Define(task)
		t.Custom = append(t.Custom, defined)
		deps = append(deps, defined)
	}

	// GitDiff is available as a standalone task.
	t.GitDiff = gitdiff.Task()

	// Create the "all" task that runs everything, then checks for uncommitted changes.
	allTask := goyek.Task{
		Name:  "all",
		Usage: "run all tasks",
		Deps:  deps,
	}
	if !cfg.SkipGitDiff {
		allTask.Action = func(a *goyek.A) {
			// Run git diff after all deps complete.
			cmd := pocket.Command(a.Context(), "git", "diff", "--exit-code")
			if err := cmd.Run(); err != nil {
				a.Fatal("uncommitted changes detected; please commit or stage your changes")
			}
		}
	}
	t.All = goyek.Define(allTask)

	return t
}
