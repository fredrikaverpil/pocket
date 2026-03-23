package pk

import (
	"context"
	"flag"
	"fmt"
	"maps"
	"slices"

	"github.com/fredrikaverpil/pocket/pk/internal/ctxkey"
	pkrun "github.com/fredrikaverpil/pocket/pk/run"
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
	// Flags defines declarative CLI flags for the task as a struct.
	// Each exported field becomes a flag. Use struct tags:
	//   `flag:"name" usage:"help text"`
	// Access flag values at runtime with [GetFlags].
	Flags any
	// Hidden makes the task invisible in CLI listings. Hidden tasks can still
	// be executed directly.
	Hidden bool
	// HideHeader suppresses the ":: taskname" header before execution.
	// Useful for tasks that output machine-readable data (e.g., JSON).
	HideHeader bool
	// Global makes the task deduplicate by name only, ignoring path.
	// Use this for install tasks that should only run once regardless of path.
	Global bool
	// Verbose forces verbose (streamed) output for this task,
	// regardless of the -v CLI flag. Can also be set via [WithVerbose].
	Verbose bool

	// flagSet is the internal FlagSet built from Flags by the engine.
	flagSet *flag.FlagSet
}

// buildFlagSet constructs the internal *flag.FlagSet from the Flags struct.
// Flags are registered in sorted order for deterministic -h output.
func (t *Task) buildFlagSet() error {
	fs, err := buildFlagSetFromStruct(t.Name, t.Flags)
	if err != nil {
		return err
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

	// Look up plan-level settings for this task (manual check, verbose, flag overrides).
	var instance *taskInstance
	if plan := planFromContext(ctx); plan != nil {
		instance = plan.taskInstanceByName(effectiveName)
	}
	if instance != nil {
		if instance.isManual && isAutoExec(ctx) {
			return nil
		}
		if instance.verbose {
			ctx = context.WithValue(ctx, ctxkey.Verbose{}, true)
		}
	}

	// Build resolved flags from declared defaults + plan overrides + CLI overrides.
	// This avoids mutating the shared flagSet, preventing races when the same
	// task runs in parallel with different WithNameSuffix variants.
	if t.Flags != nil {
		resolved, err := structToMap(t.Flags)
		if err != nil {
			return fmt.Errorf("task %q: %w", t.Name, err)
		}
		if instance != nil {
			maps.Copy(resolved, instance.flags)
		}
		if cliFlags := cliFlagsFromContext(ctx); cliFlags != nil {
			maps.Copy(resolved, cliFlags)
		}
		ctx = context.WithValue(ctx, ctxkey.TaskFlags{}, resolved)
	}

	// Check deduplication unless forceRun is set in context.
	// Deduplication uses taskID (effective name + path), or base name + "." for global tasks.
	// Global tasks use base name only (ignoring suffix) to ensure install tasks run once.
	if !forceRunFromContext(ctx) {
		tracker := executionTrackerFromContext(ctx)
		if tracker != nil {
			id := taskID{Name: effectiveName, Path: pkrun.PathFromContext(ctx)}
			if t.Global {
				id = taskID{Name: t.Name, Path: "."} // Global tasks deduplicate by base name only.
			}
			if alreadyDone := tracker.markDone(id); alreadyDone {
				return nil // Silent skip.
			}
		}
	}

	// Check if this task should run at this path based on the Plan's pathMappings.
	// This handles task-specific excludes (WithSkipTask with patterns).
	if plan := planFromContext(ctx); plan != nil {
		if info, ok := plan.pathMappings[effectiveName]; ok {
			path := pkrun.PathFromContext(ctx)
			if !slices.Contains(info.resolvedPaths, path) {
				return nil // Task is excluded from this path.
			}
		}
	}

	// Print task header before execution (unless header is hidden).
	if !t.HideHeader {
		path := pkrun.PathFromContext(ctx)
		if path != "" && path != "." {
			pkrun.Printf(ctx, ":: %s [%s]\n", effectiveName, path)
		} else {
			pkrun.Printf(ctx, ":: %s\n", effectiveName)
		}
	}

	return t.execute(ctx)
}

// execute runs the task body, recovering flagError panics from GetFlag.
func (t *Task) execute(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if fe, ok := r.(pkrun.FlagError); ok {
				err = fmt.Errorf("task %q: %w", t.Name, fe.Err)
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
