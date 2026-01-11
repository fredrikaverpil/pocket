// Package tasks provides the unified task entry point for pocket.
// It automatically creates tasks based on the provided Config.
package tasks

import (
	"context"
	"maps"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/internal/tasks/clean"
	"github.com/fredrikaverpil/pocket/internal/tasks/generate"
	"github.com/fredrikaverpil/pocket/internal/tasks/gitdiff"
	"github.com/fredrikaverpil/pocket/internal/tasks/update"
)

// Runner holds all registered tasks based on the Config.
// It orchestrates built-in tasks (all, generate, update, git-diff, clean)
// and collects user tasks for CLI registration.
type Runner struct {
	// All runs all configured tasks.
	All *pocket.Task

	// Clean removes downloaded tools and binaries.
	Clean *pocket.Task

	// Generate regenerates all generated files.
	Generate *pocket.Task

	// Update updates pocket and regenerates files.
	Update *pocket.Task

	// GitDiff fails if there are uncommitted changes.
	GitDiff *pocket.Task

	// UserTasks holds all tasks from Config.AutoRun and Config.ManualRun (for CLI registration).
	UserTasks []*pocket.Task

	// pathMappings maps task names to their PathFilter configuration.
	pathMappings map[string]*pocket.PathFilter

	// autoRunTaskNames tracks which tasks are from AutoRun (for CLI help display).
	autoRunTaskNames map[string]bool
}

// NewRunner creates a Runner based on the provided Config.
func NewRunner(cfg pocket.Config) *Runner {
	cfg = cfg.WithDefaults()
	t := &Runner{}

	// Clean is available as a standalone task.
	t.Clean = clean.Task()

	// Generate runs first - other tasks may need generated files.
	t.Generate = generate.Task(cfg)

	// Update is standalone (not part of "all").
	t.Update = update.Task(cfg)

	// GitDiff is available as a standalone task.
	t.GitDiff = gitdiff.Task()

	// Initialize maps.
	t.pathMappings = make(map[string]*pocket.PathFilter)
	t.autoRunTaskNames = make(map[string]bool)
	seenTasks := make(map[string]bool)

	// Extract tasks from AutoRun for CLI registration and execution.
	if cfg.AutoRun != nil {
		for _, task := range cfg.AutoRun.Tasks() {
			name := task.TaskName()
			if !seenTasks[name] {
				t.UserTasks = append(t.UserTasks, task)
				seenTasks[name] = true
			}
			t.autoRunTaskNames[name] = true
		}
		maps.Copy(t.pathMappings, pocket.CollectPathMappings(cfg.AutoRun))
	}

	// Extract tasks from ManualRun for CLI registration only.
	// Skip tasks already registered from AutoRun.
	for _, r := range cfg.ManualRun {
		for _, task := range r.Tasks() {
			name := task.TaskName()
			if !seenTasks[name] {
				t.UserTasks = append(t.UserTasks, task)
				seenTasks[name] = true
			}
		}
		maps.Copy(t.pathMappings, pocket.CollectPathMappings(r))
	}

	// Create the "all" task that runs everything.
	// Hidden because it's the default task (run when no task is specified).
	t.All = pocket.NewTask("all", "run all tasks", func(ctx context.Context, tc *pocket.TaskContext) error {
		exec := tc.Execution()
		// Generate first.
		if err := t.Generate.Run(ctx, exec); err != nil {
			return err
		}

		// Run the user's execution tree.
		if cfg.AutoRun != nil {
			if err := cfg.AutoRun.Run(ctx, exec); err != nil {
				return err
			}
		}

		// Git diff at the end (if not skipped).
		if !cfg.SkipGitDiff {
			return t.GitDiff.Run(ctx, exec)
		}
		return nil
	}).AsHidden()

	return t
}

// AllTasks returns all tasks including the "all" task.
// This is used by the CLI to register all available tasks.
func (t *Runner) AllTasks() []*pocket.Task {
	tasks := []*pocket.Task{t.All, t.Clean, t.Generate, t.Update, t.GitDiff}
	tasks = append(tasks, t.UserTasks...)
	return tasks
}

// PathMappings returns the path mappings for cwd-based task filtering.
// Tasks not in this map are only visible when running from the git root.
func (t *Runner) PathMappings() map[string]*pocket.PathFilter {
	return t.pathMappings
}

// AutoRunTaskNames returns the set of task names that are from AutoRun.
// Used by the CLI to display tasks in separate sections.
func (t *Runner) AutoRunTaskNames() map[string]bool {
	return t.autoRunTaskNames
}
