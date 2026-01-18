package pk

import "context"

// Plan represents the calculated execution plan.
// It holds information about all tasks that will execute.
type Plan struct {
	// Tasks is the list of tasks collected from the tree
	Tasks []*Task
}

// BuildPlan walks the Runnable tree and collects all tasks
// without executing them. This enables plan visualization and validation.
func BuildPlan(ctx context.Context, root Runnable) (*Plan, error) {
	if root == nil {
		return &Plan{}, nil
	}

	collector := &planCollector{
		tasks: make([]*Task, 0),
	}

	if err := collector.walk(ctx, root); err != nil {
		return nil, err
	}

	return &Plan{
		Tasks: collector.tasks,
	}, nil
}

// planCollector is the internal state for walking the tree
type planCollector struct {
	tasks []*Task
}

// walk recursively traverses the Runnable tree
func (pc *planCollector) walk(ctx context.Context, r Runnable) error {
	if r == nil {
		return nil
	}

	// Type switch on the concrete Runnable types
	switch v := r.(type) {
	case *Task:
		// Leaf node - collect the task
		pc.tasks = append(pc.tasks, v)

	case *serial:
		// Composition node - walk children sequentially
		for _, child := range v.runnables {
			if err := pc.walk(ctx, child); err != nil {
				return err
			}
		}

	case *parallel:
		// Composition node - walk children (order doesn't matter for collection)
		for _, child := range v.runnables {
			if err := pc.walk(ctx, child); err != nil {
				return err
			}
		}

	default:
		// Unknown runnable type - skip it
		// This allows new types to be added without breaking plan building
	}

	return nil
}
