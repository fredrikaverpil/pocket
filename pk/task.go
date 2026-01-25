package pk

import (
	"context"
	"flag"
	"fmt"
	"slices"
)

// Task represents a named, executable unit of work.
// Create tasks with NewTask.
type Task struct {
	name       string
	usage      string
	flags      *flag.FlagSet
	fn         func(context.Context) error
	hidden     bool
	manual     bool // Task only runs when explicitly invoked.
	hideHeader bool // Task runs without printing header.
	global     bool // Task deduplicates globally (ignores path).
}

// NewTask creates a new task with a Runnable body and optional CLI flags.
// Use Do() to wrap a function as a Runnable.
//
// Example with function body:
//
//	var Hello = pk.NewTask("hello", "greet", flags, pk.Do(func(ctx context.Context) error {
//	    fmt.Println("Hello!")
//	    return nil
//	}))
//
// Example with composition:
//
//	var Lint = pk.NewTask("lint", "run linters", nil, pk.Serial(Install, lintCmd()))
func NewTask(name, usage string, flags *flag.FlagSet, body Runnable) *Task {
	return &Task{
		name:  name,
		usage: usage,
		flags: flags,
		fn: func(ctx context.Context) error {
			return body.run(ctx)
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

	// Apply pre-computed flag overrides from Plan.
	// Flags are pre-merged during planning and stored on taskInstance.
	if t.flags != nil {
		if plan := PlanFromContext(ctx); plan != nil {
			if entry := plan.taskInstanceByName(effectiveName); entry != nil && entry.flags != nil {
				for name, value := range entry.flags {
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

// Hidden returns a new Task that is hidden from CLI listings.
// Hidden tasks can still be executed directly but won't appear in help.
func (t *Task) Hidden() *Task {
	return &Task{
		name:       t.name,
		usage:      t.usage,
		flags:      t.flags,
		fn:         t.fn,
		hidden:     true,
		manual:     t.manual,
		hideHeader: t.hideHeader,
		global:     t.global,
	}
}

// IsHidden returns whether the task is hidden from CLI listings.
func (t *Task) IsHidden() bool {
	return t.hidden
}

// Manual returns a new Task marked as manual.
// Manual tasks only run when explicitly invoked (e.g., `./pok hello`),
// not on bare `./pok` invocation.
func (t *Task) Manual() *Task {
	return &Task{
		name:       t.name,
		usage:      t.usage,
		flags:      t.flags,
		fn:         t.fn,
		hidden:     t.hidden,
		manual:     true,
		hideHeader: t.hideHeader,
		global:     t.global,
	}
}

// IsManual returns whether the task is manual-only.
func (t *Task) IsManual() bool {
	return t.manual
}

// HideHeader returns a new Task that runs without printing the ":: taskname" header.
// Useful for tasks that output machine-readable data (e.g., JSON).
func (t *Task) HideHeader() *Task {
	return &Task{
		name:       t.name,
		usage:      t.usage,
		flags:      t.flags,
		fn:         t.fn,
		hidden:     t.hidden,
		manual:     t.manual,
		hideHeader: true,
		global:     t.global,
	}
}

// IsHeaderHidden returns whether the task runs without printing header.
func (t *Task) IsHeaderHidden() bool {
	return t.hideHeader
}

// Global returns a new Task that deduplicates globally (by name only, ignoring path).
// Use this for install tasks and other operations that should only run once
// regardless of how many paths the parent task runs in.
func (t *Task) Global() *Task {
	return &Task{
		name:       t.name,
		usage:      t.usage,
		flags:      t.flags,
		fn:         t.fn,
		hidden:     t.hidden,
		manual:     t.manual,
		hideHeader: t.hideHeader,
		global:     true,
	}
}

// IsGlobal returns whether the task deduplicates globally.
func (t *Task) IsGlobal() bool {
	return t.global
}
