// Package tasks provides the unified task entry point for pocket.
// It automatically creates tasks based on the provided Config.
package tasks

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tasks/generate"
	"github.com/fredrikaverpil/pocket/tasks/gitdiff"
	"github.com/fredrikaverpil/pocket/tasks/update"
)

// Tasks holds all registered tasks based on the Config.
type Tasks struct {
	// All runs all configured tasks.
	All *pocket.Task

	// Generate regenerates all generated files.
	Generate *pocket.Task

	// Update updates pocket and regenerates files.
	Update *pocket.Task

	// GitDiff fails if there are uncommitted changes.
	GitDiff *pocket.Task

	// UserTasks holds all tasks from Config.Run (for CLI registration).
	UserTasks []*pocket.Task
}

// New creates tasks based on the provided Config.
func New(cfg pocket.Config) *Tasks {
	cfg = cfg.WithDefaults()
	t := &Tasks{}

	// Generate runs first - other tasks may need generated files.
	t.Generate = generate.Task(cfg)

	// Update is standalone (not part of "all").
	t.Update = update.Task(cfg)

	// GitDiff is available as a standalone task.
	t.GitDiff = gitdiff.Task()

	// Extract all tasks from the execution tree for CLI registration.
	if cfg.Run != nil {
		t.UserTasks = cfg.Run.Tasks()
	}

	// Create the "all" task that runs everything.
	t.All = &pocket.Task{
		Name:  "all",
		Usage: "run all tasks",
		Action: func(ctx context.Context, _ map[string]string) error {
			// Generate first.
			if err := t.Generate.Run(ctx); err != nil {
				return err
			}

			// Run the user's execution tree.
			if cfg.Run != nil {
				if err := cfg.Run.Run(ctx); err != nil {
					return err
				}
			}

			// Git diff at the end (if not skipped).
			if !cfg.SkipGitDiff {
				return t.GitDiff.Run(ctx)
			}
			return nil
		},
	}

	return t
}

// AllTasks returns all tasks including the "all" task.
// This is used by the CLI to register all available tasks.
func (t *Tasks) AllTasks() []*pocket.Task {
	tasks := []*pocket.Task{t.All, t.Generate, t.Update, t.GitDiff}
	tasks = append(tasks, t.UserTasks...)
	return tasks
}
