package pk

import (
	"context"
	"fmt"
)

// Task creates a named, configurable unit of work.
// Options can be provided to customize task behavior.
type Task struct {
	name    string
	options map[string]any
	fn      func(context.Context, map[string]any) error
}

// NewTask creates a new task with the given name and optional configuration function.
func NewTask(name string, fn func(context.Context, map[string]any) error) *Task {
	return &Task{
		name:    name,
		options: make(map[string]any),
		fn:      fn,
	}
}

// With adds an option to the task configuration.
// Returns the task for method chaining.
func (t *Task) With(key string, value any) *Task {
	t.options[key] = value
	return t
}

// run implements the Runnable interface.
func (t *Task) run(ctx context.Context) error {
	if t.fn == nil {
		return fmt.Errorf("task %q has no implementation", t.name)
	}
	return t.fn(ctx, t.options)
}

// Name returns the task's name (useful for plan generation and debugging).
func (t *Task) Name() string {
	return t.name
}
