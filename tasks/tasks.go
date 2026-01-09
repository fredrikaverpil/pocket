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

	// pathMappings maps task names to their PathFilter configuration.
	pathMappings map[string]*pocket.PathFilter
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
	// Also collect path mappings for cwd-based filtering.
	if cfg.Run != nil {
		t.UserTasks = cfg.Run.Tasks()
		t.pathMappings = pocket.CollectPathMappings(cfg.Run)
	} else {
		t.pathMappings = make(map[string]*pocket.PathFilter)
	}

	// Create the "all" task that runs everything.
	// Hidden because it's the default task (run when no task is specified).
	t.All = &pocket.Task{
		Name:   "all",
		Usage:  "run all tasks",
		Hidden: true,
		Action: func(ctx context.Context, _ *pocket.RunContext) error {
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

// PathMappings returns the path mappings for cwd-based task filtering.
// Tasks not in this map are only visible when running from the git root.
func (t *Tasks) PathMappings() map[string]*pocket.PathFilter {
	return t.pathMappings
}
