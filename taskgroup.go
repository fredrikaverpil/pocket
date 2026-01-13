package pocket

import "context"

// TaskGroup holds a collection of tasks with execution semantics.
// Use NewTaskGroup to create a group, then chain methods to configure it.
//
// Example (simple, parallel execution):
//
//	func Tasks() pocket.Runnable {
//	    return pocket.NewTaskGroup(FormatTask(), LintTask())
//	}
//
// Example (custom execution order):
//
//	func Tasks() pocket.Runnable {
//	    format, lint := FormatTask(), LintTask()
//	    test, vulncheck := TestTask(), VulncheckTask()
//
//	    return pocket.NewTaskGroup(format, lint, test, vulncheck).
//	        RunWith(pocket.Serial(format, lint, pocket.Parallel(test, vulncheck)))
//	}
type TaskGroup struct {
	tasks  []*Task
	runner Runnable
}

// NewTaskGroup creates a new task group with the given tasks.
// By default, tasks run in parallel. Use RunWith to customize execution order.
func NewTaskGroup(tasks ...*Task) *TaskGroup {
	return &TaskGroup{
		tasks: tasks,
	}
}

// RunWith sets a custom execution order for the task group.
// If not called, tasks run in parallel by default.
//
// Use Serial() and Parallel() to compose the execution order:
//
//	group.RunWith(pocket.Serial(format, lint, pocket.Parallel(test, vulncheck)))
func (g *TaskGroup) RunWith(r Runnable) *TaskGroup {
	g.runner = r
	return g
}

// Run executes the task group.
// If RunWith was called, uses the custom Runnable.
// Otherwise, runs all tasks in parallel.
func (g *TaskGroup) Run(ctx context.Context, exec *Execution) error {
	if g.runner != nil {
		return g.runner.Run(ctx, exec)
	}
	// Default: run all tasks in parallel.
	runnables := make([]Runnable, len(g.tasks))
	for i, t := range g.tasks {
		runnables[i] = t
	}
	return Parallel(runnables...).Run(ctx, exec)
}

// Tasks returns all tasks in the group.
func (g *TaskGroup) Tasks() []*Task {
	return g.tasks
}
