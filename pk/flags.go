package pk

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk/internal/engine"
)

// GetFlags retrieves the resolved flags for a task from context.
// It returns a struct of type T populated with the task's default values,
// any overrides from [WithFlags], and CLI flag values (highest priority).
//
// Must be called from within a task's Do function. If no flags are available
// in context, the task returns an error.
func GetFlags[T any](ctx context.Context) T {
	return engine.GetFlags[T](ctx)
}
