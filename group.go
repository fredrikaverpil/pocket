package pocket

import (
	"context"
	"reflect"

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
// Items can be *FuncDef, Runnable, or func(context.Context) error.
//
// Example:
//
//	var Lint = pocket.Func("lint", "run linter", pocket.Serial(
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
		ec.plan.PushSerial()
		defer ec.plan.Pop()
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

func (s *serial) funcs() []*FuncDef {
	all := make([]*FuncDef, 0, len(s.items))
	for _, r := range s.items {
		all = append(all, r.funcs()...)
	}
	return all
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
// Items can be *FuncDef, Runnable, or func(context.Context) error.
//
// Example:
//
//	var CI = pocket.Func("ci", "run CI", pocket.Parallel(
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
		ec.plan.PushParallel()
		defer ec.plan.Pop()
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

	g, gCtx := errgroup.WithContext(ctx)
	for i, r := range toRun {
		g.Go(func() error {
			newEC := *ec
			newEC.out = buffers[i].Output()
			newCtx := withExecContext(gCtx, &newEC)
			return r.run(newCtx)
		})
	}

	err := g.Wait()

	for _, buf := range buffers {
		buf.Flush()
	}

	return err
}

func (p *parallel) funcs() []*FuncDef {
	all := make([]*FuncDef, 0, len(p.items))
	for _, r := range p.items {
		all = append(all, r.funcs()...)
	}
	return all
}

// shouldRun checks if a runnable should run (not already executed).
// Marks it as executed if it should run.
// Thread-safe for concurrent access from parallel execution.
func shouldRun(ec *execContext, r Runnable) bool {
	key := runnableKey(r)
	return ec.dedup.shouldRun(key)
}

// runnableKey returns a unique key for deduplication.
func runnableKey(r Runnable) uintptr {
	return reflect.ValueOf(r).Pointer()
}

// toRunnables converts a slice of any to a slice of Runnable.
func toRunnables(items []any) []Runnable {
	result := make([]Runnable, 0, len(items))
	for _, item := range items {
		result = append(result, toRunnable(item))
	}
	return result
}

// toRunnable converts a single item to a Runnable.
func toRunnable(item any) Runnable {
	switch v := item.(type) {
	case Runnable:
		return v
	case func(context.Context) error:
		return &funcRunnable{fn: v}
	default:
		panic("pocket: item must be Runnable or func(context.Context) error")
	}
}

// funcRunnable wraps a plain function as a Runnable.
type funcRunnable struct {
	fn func(context.Context) error
}

func (f *funcRunnable) run(ctx context.Context) error {
	return f.fn(ctx)
}

func (f *funcRunnable) funcs() []*FuncDef {
	return nil
}

// runWithContext executes a Runnable with fresh execution context.
func runWithContext(ctx context.Context, r Runnable, out *Output, cwd string, verbose bool) error {
	ec := newExecContext(out, cwd, verbose)
	ctx = withExecContext(ctx, ec)
	return r.run(ctx)
}
