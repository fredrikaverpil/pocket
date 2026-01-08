package pocket

import (
	"context"
	"fmt"
)

// RunTask executes the given task.
// If the task has already been run, it returns the cached result.
//
// Deprecated: Use task.Run(ctx) directly instead.
func RunTask(ctx context.Context, task *Task) error {
	if task == nil {
		return fmt.Errorf("cannot run nil task")
	}
	return task.Run(ctx)
}
