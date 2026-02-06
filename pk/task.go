package pk

import (
	"context"
	"flag"
	"fmt"
	"io"
	"maps"
	"slices"
	"sort"
)

// Task represents a named, executable unit of work.
//
// Create tasks as struct literals:
//
//	var Hello = &pk.Task{
//	    Name:  "hello",
//	    Usage: "greet",
//	    Do: func(ctx context.Context) error {
//	        fmt.Println("Hello!")
//	        return nil
//	    },
//	}
//
// For composed tasks:
//
//	var Lint = &pk.Task{
//	    Name:  "lint",
//	    Usage: "run linters",
//	    Body:  pk.Serial(Install, lintCmd()),
//	}
type Task struct {
	// Name is the task's unique identifier (required).
	Name string
	// Usage is a short description shown in help output.
	Usage string
	// Do is the task's executable function. Mutually exclusive with Body.
	Do func(context.Context) error
	// Body is the task's composed logic. Mutually exclusive with Do.
	Body Runnable
	// Flags defines declarative CLI flags for the task.
	// Access flag values at runtime with [GetFlag].
	Flags map[string]FlagDef
	// Hidden makes the task invisible in CLI listings. Hidden tasks can still
	// be executed directly.
	Hidden bool
	// HideHeader suppresses the ":: taskname" header before execution.
	// Useful for tasks that output machine-readable data (e.g., JSON).
	HideHeader bool
	// Global makes the task deduplicate by name only, ignoring path.
	// Use this for install tasks that should only run once regardless of path.
	Global bool

	// flagSet is the internal FlagSet built from Flags by the engine.
	flagSet *flag.FlagSet
}

// FlagDef defines a declarative CLI flag.
// Supported Default types: string, bool, int, float64.
type FlagDef struct {
	// Default is the flag's default value.
	Default any
	// Usage is the help text for the flag.
	Usage string
}

// taskFlagsKey is the context key for resolved task flag values.
type taskFlagsKey struct{}

// withTaskFlags stores resolved flag values in context.
func withTaskFlags(ctx context.Context, flags map[string]any) context.Context {
	return context.WithValue(ctx, taskFlagsKey{}, flags)
}

// taskFlagsFromContext retrieves resolved flag values from context.
func taskFlagsFromContext(ctx context.Context) map[string]any {
	if flags, ok := ctx.Value(taskFlagsKey{}).(map[string]any); ok {
		return flags
	}
	return nil
}

// cliFlagsKey is the context key for CLI-provided flag overrides.
type cliFlagsKey struct{}

// withCLIFlags stores CLI-provided flag values in context.
func withCLIFlags(ctx context.Context, flags map[string]any) context.Context {
	return context.WithValue(ctx, cliFlagsKey{}, flags)
}

// cliFlagsFromContext retrieves CLI-provided flag values from context.
func cliFlagsFromContext(ctx context.Context) map[string]any {
	if flags, ok := ctx.Value(cliFlagsKey{}).(map[string]any); ok {
		return flags
	}
	return nil
}

// flagError is a sentinel type for GetFlag panics.
// task.run() recovers this specific type and converts it to a returned error.
type flagError struct {
	err error
}

// GetFlag retrieves a flag value from context by name.
// Panics with a flagError if the flag is not found or has a type mismatch.
// The panic is recovered by task.run() and surfaced as a returned error.
func GetFlag[T any](ctx context.Context, name string) T {
	var zero T
	flags := taskFlagsFromContext(ctx)
	if flags == nil {
		panic(flagError{fmt.Errorf("flag %q: no flags in context", name)})
	}
	v, ok := flags[name]
	if !ok {
		panic(flagError{fmt.Errorf("flag %q: not found", name)})
	}
	typed, ok := v.(T)
	if !ok {
		panic(flagError{fmt.Errorf("flag %q: expected %T, got %T", name, zero, v)})
	}
	return typed
}

// buildFlagSet constructs the internal *flag.FlagSet from the Flags map.
// Flags are registered in sorted order for deterministic -h output.
func (t *Task) buildFlagSet() error {
	fs := flag.NewFlagSet(t.Name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if len(t.Flags) == 0 {
		t.flagSet = fs
		return nil
	}

	// Sort flag names for deterministic output.
	names := make([]string, 0, len(t.Flags))
	for name := range t.Flags {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		def := t.Flags[name]
		switch v := def.Default.(type) {
		case string:
			fs.String(name, v, def.Usage)
		case bool:
			fs.Bool(name, v, def.Usage)
		case int:
			fs.Int(name, v, def.Usage)
		case float64:
			fs.Float64(name, v, def.Usage)
		default:
			return fmt.Errorf("task %q: flag %q has unsupported default type %T", t.Name, name, def.Default)
		}
	}

	t.flagSet = fs
	return nil
}

// run implements the Runnable interface.
func (t *Task) run(ctx context.Context) error {
	if t.Do == nil && t.Body == nil {
		return fmt.Errorf("task %q has no implementation", t.Name)
	}

	// Build effective name using suffix from context (e.g., "py-test:3.9").
	effectiveName := t.Name
	if suffix := nameSuffixFromContext(ctx); suffix != "" {
		effectiveName = t.Name + ":" + suffix
	}

	// Check manual status via Plan.
	if plan := PlanFromContext(ctx); plan != nil {
		if instance := plan.taskInstanceByName(effectiveName); instance != nil {
			if instance.isManual && isAutoExec(ctx) {
				return nil
			}
		}
	}

	// Build resolved flags from declared defaults + plan overrides + CLI overrides.
	// This avoids mutating the shared flagSet, preventing races when the same
	// task runs in parallel with different WithNameSuffix variants.
	if len(t.Flags) > 0 {
		resolved := make(map[string]any, len(t.Flags))
		for name, def := range t.Flags {
			resolved[name] = def.Default
		}
		// Apply plan-level overrides (from WithFlag).
		if plan := PlanFromContext(ctx); plan != nil {
			if instance := plan.taskInstanceByName(effectiveName); instance != nil {
				maps.Copy(resolved, instance.flags)
			}
		}
		// Apply CLI overrides (highest priority).
		if cliFlags := cliFlagsFromContext(ctx); cliFlags != nil {
			maps.Copy(resolved, cliFlags)
		}
		ctx = withTaskFlags(ctx, resolved)
	}

	// Check deduplication unless forceRun is set in context.
	// Deduplication uses taskID (effective name + path), or base name + "." for global tasks.
	// Global tasks use base name only (ignoring suffix) to ensure install tasks run once.
	if !forceRunFromContext(ctx) {
		tracker := executionTrackerFromContext(ctx)
		if tracker != nil {
			id := taskID{Name: effectiveName, Path: PathFromContext(ctx)}
			if t.Global {
				id = taskID{Name: t.Name, Path: "."} // Global tasks deduplicate by base name only.
			}
			if alreadyDone := tracker.markDone(id); alreadyDone {
				return nil // Silent skip.
			}
		}
	}

	// Check if this task should run at this path based on the Plan's pathMappings.
	// This handles task-specific excludes (WithExcludeTask).
	if plan := PlanFromContext(ctx); plan != nil {
		if info, ok := plan.pathMappings[effectiveName]; ok {
			path := PathFromContext(ctx)
			if !slices.Contains(info.resolvedPaths, path) {
				return nil // Task is excluded from this path.
			}
		}
	}

	// Print task header before execution (unless header is hidden).
	if !t.HideHeader {
		path := PathFromContext(ctx)
		if path != "" && path != "." {
			Printf(ctx, ":: %s [%s]\n", effectiveName, path)
		} else {
			Printf(ctx, ":: %s\n", effectiveName)
		}
	}

	return t.execute(ctx)
}

// execute runs the task body, recovering flagError panics from GetFlag.
func (t *Task) execute(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if fe, ok := r.(flagError); ok {
				err = fmt.Errorf("task %q: %w", t.Name, fe.err)
			} else {
				panic(r) // Re-panic for unrelated panics.
			}
		}
	}()
	if t.Do != nil {
		return t.Do(ctx)
	}
	return t.Body.run(ctx)
}
