package pk

import (
	"context"
	"sync"
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

	// Check if context is already canceled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(p.runnables))

	// Start all runnables concurrently
	for _, r := range p.runnables {
		// Check context before starting each goroutine
		select {
		case <-ctx.Done():
			// Context canceled, don't start more goroutines
			// Wait for already-started ones to finish
			wg.Wait()
			close(errChan)
			// Return context error or first runnable error
			select {
			case err := <-errChan:
				return err
			default:
				return ctx.Err()
			}
		default:
		}

		wg.Add(1)
		go func(runnable Runnable) {
			defer wg.Done()
			if err := runnable.run(ctx); err != nil {
				errChan <- err
			}
		}(r)
	}

	// Wait for all to complete
	wg.Wait()
	close(errChan)

	// Return first error if any
	for err := range errChan {
		return err
	}

	return nil
}
