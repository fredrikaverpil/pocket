// Package tasks provides the unified task entry point for pocket.
// It automatically creates tasks based on the provided Config.
package tasks

import (
	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tasks/generate"
	"github.com/fredrikaverpil/pocket/tasks/gitdiff"
	"github.com/fredrikaverpil/pocket/tasks/update"
	"github.com/goyek/goyek/v3"
)

// Tasks holds all registered tasks based on the Config.
type Tasks struct {
	// All runs all configured tasks.
	All *goyek.DefinedTask

	// Generate regenerates all generated files.
	Generate *goyek.DefinedTask

	// Update updates pocket and regenerates files.
	Update *goyek.DefinedTask

	// GitDiff fails if there are uncommitted changes.
	GitDiff *goyek.DefinedTask

	// Tasks holds standalone tasks registered for this context.
	Tasks []*goyek.DefinedTask

	// TaskGroupTasks holds all tasks from registered task groups.
	TaskGroupTasks []*goyek.DefinedTask
}

// New creates tasks based on the provided Config.
// It filters both config and task groups for the current context.
func New(cfg pocket.Config, context string) *Tasks {
	cfg = cfg.WithDefaults()
	t := &Tasks{}

	// Generate runs first - other tasks may need generated files.
	t.Generate = generate.Task(cfg)

	// Update is standalone (not part of "all")
	t.Update = update.Task(cfg)

	// Start with generate as first dep (runs before everything else)
	deps := goyek.Deps{t.Generate}

	// Filter config for context (this also filters task groups).
	filteredCfg := cfg.ForContext(context)

	// Create tasks from context-filtered task groups.
	for _, tg := range filteredCfg.TaskGroups {
		tgTasks := tg.Tasks(filteredCfg)
		t.TaskGroupTasks = append(t.TaskGroupTasks, tgTasks...)
		for _, task := range tgTasks {
			deps = append(deps, task)
		}
	}

	// Define standalone tasks from filtered config and add them to deps.
	for _, task := range filteredCfg.GetTasks() {
		defined := goyek.Define(task)
		t.Tasks = append(t.Tasks, defined)
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
