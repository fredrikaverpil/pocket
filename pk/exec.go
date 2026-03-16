package pk

import "context"

// Do wraps a Go function as a [Runnable] for use in task composition.
//
//	pk.Do(func(ctx context.Context) error {
//	    return run.Exec(ctx, "golangci-lint", "run", "--fix", "./...")
//	})
func Do(fn func(ctx context.Context) error) Runnable {
	return &doRunnable{fn: fn}
}

// doRunnable wraps a function as a Runnable.
type doRunnable struct {
	fn func(ctx context.Context) error
}

func (d *doRunnable) run(ctx context.Context) error {
	return d.fn(ctx)
}
