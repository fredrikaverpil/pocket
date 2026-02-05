package pk

import (
	"context"
	"flag"
	"fmt"
	"io"
	"slices"
)

// Task represents a named, executable unit of work.
// Create tasks with [NewTask].
type Task struct {
	name       string
	usage      string
	flags      *flag.FlagSet
	fn         func(context.Context) error
	hidden     bool
	hideHeader bool // Task runs without printing header.
	global     bool // Task deduplicates globally (ignores path).
}

// TaskConfig configures a [Task] created by [NewTask].
//
// Name and Body are required. All other fields are optional and use
// zero-value defaults (no flags, visible, local deduplication, header shown).
type TaskConfig struct {
	// Name is the task's unique identifier (required).
	Name string
	// Usage is a short description shown in help output.
	Usage string
	// Body is the task's executable logic (required). Use [Do] to wrap a function.
	Body Runnable
	// Flags defines CLI flags for the task. If nil, an empty FlagSet is created.
	Flags *flag.FlagSet
	// Hidden makes the task invisible in CLI listings. Hidden tasks can still
	// be executed directly.
	Hidden bool
	// Global makes the task deduplicate by name only, ignoring path.
	// Use this for install tasks that should only run once regardless of path.
	Global bool
	// HideHeader suppresses the ":: taskname" header before execution.
	// Useful for tasks that output machine-readable data (e.g., JSON).
	HideHeader bool
}

// NewTask creates a new [Task] from the given [TaskConfig].
//
// Example with function body:
//
//	var Hello = pk.NewTask(pk.TaskConfig{
//	    Name:  "hello",
//	    Usage: "greet",
//	    Body: pk.Do(func(ctx context.Context) error {
//	        fmt.Println("Hello!")
//	        return nil
//	    }),
//	})
//
// Example with composition:
//
//	var Lint = pk.NewTask(pk.TaskConfig{
//	    Name:  "lint",
//	    Usage: "run linters",
//	    Body:  pk.Serial(Install, lintCmd()),
//	})
//
// Example install task (hidden, global deduplication):
//
//	var Install = pk.NewTask(pk.TaskConfig{
//	    Name:   "install:tool",
//	    Usage:  "install tool",
//	    Body:   installBody,
//	    Hidden: true,
//	    Global: true,
//	})
func NewTask(cfg TaskConfig) *Task {
	flags := cfg.Flags
	if flags == nil {
		flags = flag.NewFlagSet(cfg.Name, flag.ContinueOnError)
	}
	// Suppress default flag.Usage output; we use printTaskHelp for custom help.
	flags.SetOutput(io.Discard)
	return &Task{
		name:       cfg.Name,
		usage:      cfg.Usage,
		flags:      flags,
		hidden:     cfg.Hidden,
		hideHeader: cfg.HideHeader,
		global:     cfg.Global,
		fn: func(ctx context.Context) error {
			return cfg.Body.run(ctx)
		},
	}
}

// run implements the Runnable interface.
func (t *Task) run(ctx context.Context) error {
	if t.fn == nil {
		return fmt.Errorf("task %q has no implementation", t.name)
	}

	// Build effective name using suffix from context (e.g., "py-test:3.9").
	effectiveName := t.name
	if suffix := nameSuffixFromContext(ctx); suffix != "" {
		effectiveName = t.name + ":" + suffix
	}

	// Apply pre-computed flag overrides from Plan and check manual status.
	// Flags are pre-merged during planning and stored on taskInstance.
	if plan := PlanFromContext(ctx); plan != nil {
		if instance := plan.taskInstanceByName(effectiveName); instance != nil {
			// Skip manual tasks during auto execution.
			if instance.isManual && isAutoExec(ctx) {
				return nil
			}
			if t.flags != nil && instance.flags != nil {
				for name, value := range instance.flags {
					if f := t.flags.Lookup(name); f != nil {
						if err := f.Value.Set(fmt.Sprint(value)); err != nil {
							return fmt.Errorf("task %q: setting flag %q to %v: %w", effectiveName, name, value, err)
						}
					}
				}
			}
		}
	}

	// Check deduplication unless forceRun is set in context.
	// Deduplication uses taskID (effective name + path), or base name + "." for global tasks.
	// Global tasks use base name only (ignoring suffix) to ensure install tasks run once.
	if !forceRunFromContext(ctx) {
		tracker := executionTrackerFromContext(ctx)
		if tracker != nil {
			id := taskID{Name: effectiveName, Path: PathFromContext(ctx)}
			if t.global {
				id = taskID{Name: t.name, Path: "."} // Global tasks deduplicate by base name only.
			}
			if alreadyDone := tracker.markDone(id); alreadyDone {
				return nil // Silent skip.
			}
		}
	}

	// Check if this task should run at this path based on the Plan's pathMappings.
	// This handles task-specific excludes (WithExcludeTask).
	if plan := PlanFromContext(ctx); plan != nil {
		if info, ok := plan.pathMappings[effectiveName]; ok {
			path := PathFromContext(ctx)
			if !slices.Contains(info.resolvedPaths, path) {
				return nil // Task is excluded from this path.
			}
		}
	}

	// Print task header before execution (unless header is hidden).
	if !t.hideHeader {
		path := PathFromContext(ctx)
		if path != "" && path != "." {
			Printf(ctx, ":: %s [%s]\n", effectiveName, path)
		} else {
			Printf(ctx, ":: %s\n", effectiveName)
		}
	}

	return t.fn(ctx)
}

// Name returns the task's name (useful for plan generation and debugging).
func (t *Task) Name() string {
	return t.name
}

// Usage returns the task's usage description.
func (t *Task) Usage() string {
	return t.usage
}

// Flags returns the task's FlagSet, or nil if no flags are defined.
func (t *Task) Flags() *flag.FlagSet {
	return t.flags
}

// IsHidden returns whether the task is hidden from CLI listings.
func (t *Task) IsHidden() bool {
	return t.hidden
}

// IsHeaderHidden returns whether the task runs without printing header.
func (t *Task) IsHeaderHidden() bool {
	return t.hideHeader
}

// IsGlobal returns whether the task deduplicates globally.
func (t *Task) IsGlobal() bool {
	return t.global
}
