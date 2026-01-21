package pk

import (
	"context"
	"flag"
	"fmt"
)

// Task represents a named, executable unit of work.
// Create tasks with NewTask.
type Task struct {
	name   string
	usage  string
	flags  *flag.FlagSet
	fn     func(context.Context) error
	hidden bool
	manual bool // Task only runs when explicitly invoked.
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

	// Check deduplication unless forceRun is set in context.
	// Deduplication is by (task name, path) tuple.
	if !forceRunFromContext(ctx) {
		tracker := executionTrackerFromContext(ctx)
		if tracker != nil {
			path := PathFromContext(ctx)
			if alreadyDone := tracker.markDone(t.name, path); alreadyDone {
				return nil // Silent skip.
			}
		}
	}

	// Print task header before execution.
	Printf(ctx, ":: %s\n", t.name)

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
		name:   t.name,
		usage:  t.usage,
		flags:  t.flags,
		fn:     t.fn,
		hidden: true,
		manual: t.manual,
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
		name:   t.name,
		usage:  t.usage,
		flags:  t.flags,
		fn:     t.fn,
		hidden: t.hidden,
		manual: true,
	}
}

// IsManual returns whether the task is manual-only.
func (t *Task) IsManual() bool {
	return t.manual
}
