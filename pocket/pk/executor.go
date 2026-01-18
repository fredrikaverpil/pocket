package pk

import "context"

// Execute runs a Runnable with the given context.
// This is the bridge function that allows external packages
// to execute Runnables without exposing the run() method.
func Execute(ctx context.Context, r Runnable) error {
	if r == nil {
		return nil
	}
	return r.run(ctx)
}
