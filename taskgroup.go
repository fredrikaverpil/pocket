package pocket

import "context"

// TaskGroup holds a collection of tasks with execution and detection semantics.
// Use NewTaskGroup to create a group, then chain methods to configure it.
//
// Example (simple, parallel execution):
//
//	func Tasks() pocket.Runnable {
//	    return pocket.NewTaskGroup(FormatTask(), LintTask()).
//	        DetectBy(pocket.DetectByFile("go.mod"))
//	}
//
// Example (custom execution order):
//
//	func Tasks() pocket.Runnable {
//	    format, lint := FormatTask(), LintTask()
//	    test, vulncheck := TestTask(), VulncheckTask()
//
//	    return pocket.NewTaskGroup(format, lint, test, vulncheck).
//	        RunWith(func(ctx context.Context) error {
//	            if err := pocket.Serial(format, lint).Run(ctx); err != nil {
//	                return err
//	            }
//	            return pocket.Parallel(test, vulncheck).Run(ctx)
//	        }).
//	        DetectBy(pocket.DetectByFile("go.mod"))
//	}
type TaskGroup struct {
	tasks    []*Task
	runFn    func(context.Context) error
	detectFn func() []string
}

// NewTaskGroup creates a new task group with the given tasks.
// By default, tasks run in parallel. Use RunWith to customize execution order.
func NewTaskGroup(tasks ...*Task) *TaskGroup {
	return &TaskGroup{
		tasks: tasks,
	}
}

// RunWith sets a custom execution function for the task group.
// If not called, tasks run in parallel by default.
//
// The function receives the context and should orchestrate task execution
// using Serial() and Parallel() as needed.
func (g *TaskGroup) RunWith(fn func(context.Context) error) *TaskGroup {
	g.runFn = fn
	return g
}

// DetectBy sets the detection function for auto-detection.
// This makes the TaskGroup implement Detectable.
//
// Example:
//
//	DetectBy(pocket.DetectByFile("go.mod"))
//	DetectBy(pocket.DetectByExtension(".py"))
//	DetectBy(func() []string { return []string{"."} })
func (g *TaskGroup) DetectBy(fn func() []string) *TaskGroup {
	g.detectFn = fn
	return g
}

// Run executes the task group.
// If RunWith was called, uses the custom function.
// Otherwise, runs all tasks in parallel.
func (g *TaskGroup) Run(ctx context.Context) error {
	if g.runFn != nil {
		return g.runFn(ctx)
	}
	// Default: run all tasks in parallel.
	runnables := make([]Runnable, len(g.tasks))
	for i, t := range g.tasks {
		runnables[i] = t
	}
	return Parallel(runnables...).Run(ctx)
}

// Tasks returns all tasks in the group.
func (g *TaskGroup) Tasks() []*Task {
	return g.tasks
}

// DefaultDetect returns the detection function.
// Implements the Detectable interface.
func (g *TaskGroup) DefaultDetect() func() []string {
	return g.detectFn
}

// DetectByFile is a convenience method that detects directories containing
// any of the specified files (e.g., "go.mod", "pyproject.toml").
func (g *TaskGroup) DetectByFile(filenames ...string) *TaskGroup {
	return g.DetectBy(func() []string { return DetectByFile(filenames...) })
}

// DetectByExtension is a convenience method that detects directories containing
// files with any of the specified extensions (e.g., ".py", ".md").
func (g *TaskGroup) DetectByExtension(extensions ...string) *TaskGroup {
	return g.DetectBy(func() []string { return DetectByExtension(extensions...) })
}
