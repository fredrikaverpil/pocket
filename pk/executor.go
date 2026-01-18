package pk

import (
	"context"
)

// Execute runs a Runnable with the given context.
// This is the bridge function that allows external packages
// to execute Runnables without exposing the run() method.
// It builds a plan first, then executes with the plan in context
// to avoid re-resolving paths during execution.
func Execute(ctx context.Context, r Runnable) error {
	if r == nil {
		return nil
	}

	// Build plan once
	p, err := NewPlan(ctx, r)
	if err != nil {
		return err
	}

	// Execute with plan in context
	ctx = withPlan(ctx, p)
	return r.run(ctx)
}
