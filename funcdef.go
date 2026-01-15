package pocket

import (
	"context"
)

// FuncDef represents a named function that can be executed.
// Create with pocket.Func() - this is the only way to create runnable functions.
//
// The body can be either:
//   - A plain function: func(context.Context) error
//   - A Runnable composition: pocket.Serial(...) or pocket.Parallel(...)
//
// Example:
//
//	// Simple function
//	var Format = pocket.Func("go-format", "format Go code", func(ctx context.Context) error {
//	    return pocket.Exec(ctx, "go", "fmt", "./...")
//	})
//
//	// Function with dependencies
//	var Lint = pocket.Func("go-lint", "run linter", pocket.Serial(
//	    InstallLinter,
//	    func(ctx context.Context) error {
//	        return pocket.Exec(ctx, "golangci-lint", "run", "./...")
//	    },
//	))
//
//	// Hidden functions (e.g., tool installers)
//	var InstallLinter = pocket.Func("install:linter", "install linter", install).Hidden()
type FuncDef struct {
	name   string
	usage  string
	body   Runnable
	opts   any
	hidden bool
}

// Func creates a new function definition.
// This is the only way to create functions that can be used with Serial/Parallel.
//
// The name is used for CLI commands (e.g., "go-format" becomes ./pok go-format).
// The usage is displayed in help output.
// The body can be:
//   - func(context.Context) error - a plain function
//   - Runnable - a Serial/Parallel composition
func Func(name, usage string, body any) *FuncDef {
	if name == "" {
		panic("pocket.Func: name is required")
	}
	if usage == "" {
		panic("pocket.Func: usage is required")
	}
	if body == nil {
		panic("pocket.Func: body is required")
	}

	return &FuncDef{
		name:  name,
		usage: usage,
		body:  toRunnable(body),
	}
}

// With returns a copy with options attached.
// Options are accessible via pocket.Options[T](ctx) in the function.
//
// Example:
//
//	type FormatOptions struct {
//	    Config string
//	}
//
//	var Format = pocket.Func("format", "format code", formatImpl).
//	    With(FormatOptions{Config: ".golangci.yml"})
//
//	func formatImpl(ctx context.Context) error {
//	    opts := pocket.Options[FormatOptions](ctx)
//	    // use opts.Config
//	}
func (f *FuncDef) With(opts any) *FuncDef {
	cp := *f
	cp.opts = opts
	return &cp
}

// Hidden returns a copy marked as hidden from CLI help.
// Hidden functions can still be executed but don't appear in ./pok -h.
// Use this for internal functions like tool installers.
func (f *FuncDef) Hidden() *FuncDef {
	cp := *f
	cp.hidden = true
	return &cp
}

// Name returns the function's CLI name.
func (f *FuncDef) Name() string {
	return f.name
}

// Usage returns the function's help text.
func (f *FuncDef) Usage() string {
	return f.usage
}

// IsHidden returns whether the function is hidden from CLI help.
func (f *FuncDef) IsHidden() bool {
	return f.hidden
}

// Opts returns the function's options, or nil if none.
func (f *FuncDef) Opts() any {
	return f.opts
}

// Run executes this function with the given context.
// This is useful for testing or programmatic execution.
func (f *FuncDef) Run(ctx context.Context) error {
	return f.run(ctx)
}

// run executes this function with the given context.
// This is called by the framework - users should not call this directly.
func (f *FuncDef) run(ctx context.Context) error {
	ec := getExecContext(ctx)

	// In collect mode, register function and collect nested deps from static tree
	if ec.mode == modeCollect {
		// Check if this would be deduplicated
		deduped := !ec.dedup.shouldRun(runnableKey(f))
		ec.plan.AddFunc(f.name, f.usage, f.hidden, deduped)
		defer ec.plan.PopFunc()

		// Only recurse into Runnable body - do NOT call function bodies
		// This ensures collection is side-effect free and only sees static composition
		if f.body != nil {
			return f.body.run(ctx)
		}
		// Plain functions are wrapped as funcRunnable and not called during collection
		return nil
	}

	// Execute mode - print task header
	printTaskHeader(ctx, f.name)

	// Inject options into context if present
	if f.opts != nil {
		ctx = withOptions(ctx, f.opts)
	}

	// Execute the Runnable body
	return f.body.run(ctx)
}

// funcs returns all named functions in this definition's dependency tree.
// For a plain function, returns just itself.
// For a Runnable body, traverses the tree to collect all FuncDefs.
func (f *FuncDef) funcs() []*FuncDef {
	var result []*FuncDef

	// Include self if not hidden
	if !f.hidden {
		result = append(result, f)
	}

	// If body is a Runnable, collect its nested FuncDefs
	if f.body != nil {
		result = append(result, f.body.funcs()...)
	}

	return result
}

// Runnable is the interface for anything that can be executed.
// It uses unexported methods to prevent external implementation,
// ensuring only pocket types (FuncDef, serial, parallel, PathFilter) can satisfy it.
//
// Users create Runnables via:
//   - pocket.Func() for individual functions
//   - pocket.Serial() for sequential execution
//   - pocket.Parallel() for concurrent execution
//   - pocket.Paths() for path filtering
type Runnable interface {
	run(ctx context.Context) error
	funcs() []*FuncDef
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
