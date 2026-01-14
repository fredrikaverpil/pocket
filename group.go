package pocket

import (
	"context"
	"reflect"

	"golang.org/x/sync/errgroup"
)

// depsError wraps an error from Serial/Parallel execution mode.
// The framework recovers this panic and converts it to an error.
type depsError struct {
	err error
}

// serial executes items sequentially.
type serial struct {
	items []Runnable
}

// Serial has two modes based on whether the first argument is a context.Context:
//
// Execution mode (first arg is context.Context):
//
//	pocket.Serial(ctx, install1, install2)
//
// Executes items sequentially with deduplication. Panics on error (framework recovers).
//
// Composition mode (first arg is NOT context.Context):
//
//	pocket.Serial(task1, task2, task3)
//
// Returns a Runnable for use in Config or nested composition.
//
// Items can be *FuncDef, Runnable, or func(context.Context) error.
func Serial(items ...any) Runnable {
	if len(items) == 0 {
		return &serial{items: nil}
	}

	// Check if first item is context.Context (execution mode)
	if ctx, ok := items[0].(context.Context); ok {
		if err := executeSerial(ctx, items[1:]); err != nil {
			panic(depsError{err})
		}
		return nil
	}

	// Composition mode - return Runnable
	return &serial{items: toRunnables(items)}
}

// executeSerial runs items sequentially with deduplication.
func executeSerial(ctx context.Context, items []any) error {
	ec := getExecContext(ctx)
	for _, item := range items {
		r := toRunnable(item)
		if !shouldRun(ec, r) {
			continue
		}
		if err := r.run(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *serial) run(ctx context.Context) error {
	ec := getExecContext(ctx)
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

// Parallel has two modes based on whether the first argument is a context.Context:
//
// Execution mode (first arg is context.Context):
//
//	pocket.Parallel(ctx, test1, test2)
//
// Executes items concurrently with deduplication. Panics on error (framework recovers).
//
// Composition mode (first arg is NOT context.Context):
//
//	pocket.Parallel(task1, task2)
//
// Returns a Runnable for use in Config or nested composition.
//
// Items can be *FuncDef, Runnable, or func(context.Context) error.
func Parallel(items ...any) Runnable {
	if len(items) == 0 {
		return &parallel{items: nil}
	}

	// Check if first item is context.Context (execution mode)
	if ctx, ok := items[0].(context.Context); ok {
		if err := executeParallel(ctx, items[1:]); err != nil {
			panic(depsError{err})
		}
		return nil
	}

	// Composition mode - return Runnable
	return &parallel{items: toRunnables(items)}
}

// executeParallel runs items concurrently with deduplication.
func executeParallel(ctx context.Context, items []any) error {
	ec := getExecContext(ctx)

	var toRun []Runnable
	for _, item := range items {
		r := toRunnable(item)
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

func (p *parallel) run(ctx context.Context) error {
	if len(p.items) == 0 {
		return nil
	}

	ec := getExecContext(ctx)

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

// Run executes a Runnable with fresh execution context.
func Run(ctx context.Context, r Runnable, out *Output, cwd string, verbose bool) error {
	ec := newExecContext(out, cwd, verbose)
	ctx = withExecContext(ctx, ec)
	return r.run(ctx)
}
