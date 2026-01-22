package pk

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Runnable represents a unit of execution in the task graph.
// The run method is intentionally unexported to keep execution internals private.
type Runnable interface {
	run(ctx context.Context) error
}

// Serial composes multiple runnables to execute sequentially.
// Execution stops on the first error.
func Serial(runnables ...Runnable) Runnable {
	return &serial{runnables: runnables}
}

// Parallel composes multiple runnables to execute concurrently.
// All runnables are started simultaneously, and execution waits for all to complete.
// Returns the first error encountered, or nil if all succeed.
func Parallel(runnables ...Runnable) Runnable {
	return &parallel{runnables: runnables}
}

// serial is the internal implementation of sequential composition.
type serial struct {
	runnables []Runnable
}

func (s *serial) run(ctx context.Context) error {
	for _, r := range s.runnables {
		if err := r.run(ctx); err != nil {
			return err
		}
	}
	return nil
}

// parallel is the internal implementation of concurrent composition.
type parallel struct {
	runnables []Runnable
}

func (p *parallel) run(ctx context.Context) error {
	if len(p.runnables) == 0 {
		return nil
	}

	// Check if context is already canceled.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Single item? Run directly without buffering.
	if len(p.runnables) == 1 {
		return p.runnables[0].run(ctx)
	}

	// Multiple items: use errgroup and buffered output.
	// Deduplication is handled by Task.run() - no pre-filtering needed.
	parentOut := OutputFromContext(ctx)
	buffers := make([]*bufferedOutput, len(p.runnables))
	for i := range p.runnables {
		buffers[i] = newBufferedOutput(parentOut)
	}
	var flushMu sync.Mutex

	g, gCtx := errgroup.WithContext(ctx)
	for i, r := range p.runnables {
		g.Go(func() error {
			childCtx := WithOutput(gCtx, buffers[i].Output())
			err := r.run(childCtx)

			// Flush immediately on completion (first-to-complete flushes first).
			flushMu.Lock()
			buffers[i].Flush()
			flushMu.Unlock()

			return err
		})
	}
	return g.Wait()
}
