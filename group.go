package pocket

import (
	"context"
	"reflect"
	"sync"

	"golang.org/x/sync/errgroup"
)

// serial executes items sequentially.
type serial struct {
	items []Runnable
}

// Serial composes items to run sequentially.
//
// Returns a Runnable that executes items in order. Use it to:
//   - Define dependencies: Serial(Install, TaskImpl)
//   - Compose tasks in Config: Serial(Format, Lint, Test)
//
// Items can be *TaskDef, Runnable, or func(context.Context) error.
//
// Example:
//
//	var Lint = pocket.Task("lint", "run linter", pocket.Serial(
//	    golangcilint.Install,
//	    func(ctx context.Context) error {
//	        return pocket.Exec(ctx, "golangci-lint", "run", "./...")
//	    },
//	))
func Serial(items ...any) Runnable {
	return &serial{items: toRunnables(items)}
}

func (s *serial) run(ctx context.Context) error {
	ec := getExecContext(ctx)

	// In collect mode, register structure and recurse
	if ec.mode == modeCollect {
		ec.plan.pushSerial()
		defer ec.plan.pop()
		for _, r := range s.items {
			if err := r.run(ctx); err != nil {
				return err
			}
		}
		return nil
	}

	// Execute mode - run with deduplication
	for _, r := range s.items {
		if !shouldRun(ec, r) {
			continue
		}
		if err := r.run(ctx); err != nil {
			return err
		}
	}
	return nil
}


// parallel executes items concurrently.
type parallel struct {
	items []Runnable
}

// Parallel composes items to run concurrently.
//
// Returns a Runnable that executes items in parallel. Use it to:
//   - Run independent tasks concurrently: Parallel(Lint, Test)
//   - Compose in Config: Parallel(task1, task2)
//
// Items can be *TaskDef, Runnable, or func(context.Context) error.
//
// Example:
//
//	var CI = pocket.Task("ci", "run CI", pocket.Parallel(
//	    Lint,
//	    Test,
//	))
func Parallel(items ...any) Runnable {
	return &parallel{items: toRunnables(items)}
}

func (p *parallel) run(ctx context.Context) error {
	if len(p.items) == 0 {
		return nil
	}

	ec := getExecContext(ctx)

	// In collect mode, register structure and recurse (sequentially for simplicity)
	if ec.mode == modeCollect {
		ec.plan.pushParallel()
		defer ec.plan.pop()
		for _, r := range p.items {
			if err := r.run(ctx); err != nil {
				return err
			}
		}
		return nil
	}

	// Execute mode - run concurrently with deduplication
	var toRun []Runnable
	for _, r := range p.items {
		if shouldRun(ec, r) {
			toRun = append(toRun, r)
		}
	}

	if len(toRun) == 0 {
		return nil
	}
	if len(toRun) == 1 {
		return toRun[0].run(ctx)
	}

	buffers := make([]*bufferedOutput, len(toRun))
	for i := range buffers {
		buffers[i] = newBufferedOutput(ec.out)
	}

	var flushMu sync.Mutex

	g, gCtx := errgroup.WithContext(ctx)
	for i, r := range toRun {
		g.Go(func() error {
			newEC := *ec
			newEC.out = buffers[i].Output()
			newCtx := withExecContext(gCtx, &newEC)
			err := r.run(newCtx)

			flushMu.Lock()
			buffers[i].Flush()
			flushMu.Unlock()

			return err
		})
	}

	return g.Wait()
}


// shouldRun checks if a runnable should run (not already executed).
// Marks it as executed if it should run.
// Thread-safe for concurrent access from parallel execution.
//
// Only TaskDef is deduplicated - inner runnables (doRunnable, commandRunnable, etc.)
// always run because they represent the actual work that their parent task performs.
// Without this, Clone(task, Opts(...)) variants would incorrectly skip work because
// they share the same inner runnable pointers.
func shouldRun(ec *execContext, r Runnable) bool {
	// Only deduplicate TaskDef - inner runnables always run
	if _, ok := r.(*TaskDef); !ok {
		return true
	}
	key := runnableKey(r)
	return ec.dedup.shouldRun(key)
}

// runnableKey returns a unique key for deduplication.
func runnableKey(r Runnable) uintptr {
	return reflect.ValueOf(r).Pointer()
}

// runWithContext executes a Runnable with fresh execution context.
func runWithContext(ctx context.Context, r Runnable, out *Output, cwd string, verbose bool) error {
	ec := newExecContext(out, cwd, verbose)
	ctx = withExecContext(ctx, ec)
	return r.run(ctx)
}
