package pk

import (
	"context"
	"flag"
	"fmt"
)

// Task represents a named, executable unit of work.
// Create tasks with NewTask or NewTaskWithFlags.
type Task struct {
	name   string
	usage  string
	flags  *flag.FlagSet
	fn     func(context.Context) error
	hidden bool
}

// NewTask creates a new task without CLI flags.
func NewTask(name, usage string, fn func(context.Context) error) *Task {
	return &Task{
		name:  name,
		usage: usage,
		fn:    fn,
	}
}

// NewTaskWithFlags creates a new task with CLI flags.
// The FlagSet should be created with flag.ContinueOnError for proper error handling.
func NewTaskWithFlags(name, usage string, flags *flag.FlagSet, fn func(context.Context) error) *Task {
	return &Task{
		name:  name,
		usage: usage,
		flags: flags,
		fn:    fn,
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
	}
}

// IsHidden returns whether the task is hidden from CLI listings.
func (t *Task) IsHidden() bool {
	return t.hidden
}
