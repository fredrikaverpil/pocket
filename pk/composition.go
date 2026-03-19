package pk

import (
	"context"
	"sync"

	"github.com/fredrikaverpil/pocket/pk/internal/ctxkey"
	pkrun "github.com/fredrikaverpil/pocket/pk/run"
	"golang.org/x/sync/errgroup"
)

// Runnable is the core abstraction for composable units of work.
// Tasks, Serial, Parallel, and WithOptions all produce Runnables that
// can be nested to build execution trees.
//
// The run method is unexported; users compose Runnables via the
// provided constructors rather than implementing the interface directly.
type Runnable interface {
	run(ctx context.Context) error
}

// Serial composes multiple runnables to execute sequentially.
// Execution stops on the first error.
func Serial(runnables ...Runnable) Runnable {
	return &serial{runnables: runnables}
}

// Parallel composes multiple runnables to execute concurrently with buffered output.
// Each runnable's output is captured and flushed atomically on completion to prevent
// interleaving. If any runnable fails, the shared context is cancelled and remaining
// runnables exit early. A single runnable runs without buffering for real-time output.
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
	parentOut := pkrun.OutputFromContext(ctx)
	if parentOut == nil {
		parentOut = pkrun.StdOutput()
	}
	buffers := make([]*bufferedOutput, len(p.runnables))
	for i := range p.runnables {
		buffers[i] = newBufferedOutput(parentOut)
	}
	var flushMu sync.Mutex

	g, gCtx := errgroup.WithContext(ctx)
	for i, r := range p.runnables {
		g.Go(func() error {
			childCtx := context.WithValue(gCtx, ctxkey.Output{}, buffers[i].output())
			err := r.run(childCtx)

			// Flush immediately on completion (first-to-complete flushes first).
			flushMu.Lock()
			buffers[i].flush()
			flushMu.Unlock()

			return err
		})
	}
	return g.Wait()
}
