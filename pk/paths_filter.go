package pk

import (
	"context"
	"os"
	"path/filepath"
)

// PathOption configures path filtering and execution behavior for a Runnable.
// Path options determine which directories a task should execute in and
// control deduplication behavior.
type PathOption func(*pathFilter)

// DetectFunc is a function that filters directories to find those with specific markers.
// It receives the pre-walked directory list and git root path, and returns matching directories.
// Used with WithDetect to dynamically discover paths for task execution.
type DetectFunc func(dirs []string, gitRoot string) []string

// WithIncludePath adds include patterns for path filtering.
// Only directories matching any of the patterns will be included.
// Patterns are relative to the git root and are interpreted as regular expressions.
func WithIncludePath(patterns ...string) PathOption {
	return func(pf *pathFilter) {
		pf.includePaths = append(pf.includePaths, patterns...)
	}
}

// WithExcludePath adds exclude patterns for path filtering.
// Directories matching any of the patterns will be excluded for ALL tasks in the current scope.
// Patterns are relative to the git root and are interpreted as regular expressions.
func WithExcludePath(patterns ...string) PathOption {
	return func(pf *pathFilter) {
		for _, p := range patterns {
			pf.excludePaths = append(pf.excludePaths, excludePattern{
				pattern: p,
				tasks:   nil, // Global for the scope
			})
		}
	}
}

// WithExcludeTask excludes a specific task from certain patterns.
// Patterns are relative to the git root and are interpreted as regular expressions.
// The task can be specified by its string name or by the task object itself.
func WithExcludeTask(task any, patterns ...string) PathOption {
	return func(pf *pathFilter) {
		name := toTaskName(task)
		if name == "" {
			return
		}
		for _, p := range patterns {
			pf.excludePaths = append(pf.excludePaths, excludePattern{
				pattern: p,
				tasks:   []string{name},
			})
		}
	}
}

// WithSkipTask completely removes specific tasks from the current scope.
// Tasks can be specified by their string name or by the task object itself.
func WithSkipTask(tasks ...any) PathOption {
	return func(pf *pathFilter) {
		pf.skippedTasks = append(pf.skippedTasks, toTaskNames(tasks)...)
	}
}

// WithFlag sets a default flag value for a specific task in the current scope.
// The task can be specified by its string name or by the task object itself.
func WithFlag(task any, flagName string, value any) PathOption {
	return func(pf *pathFilter) {
		if name := toTaskName(task); name != "" {
			pf.flags = append(pf.flags, flagOverride{
				taskName: name,
				flagName: flagName,
				value:    value,
			})
		}
	}
}

// WithForceRun disables task deduplication for the wrapped Runnable.
// By default, tasks are deduplicated per (task pointer, path) pair within
// a single invocation. WithForceRun causes the task to always execute,
// even if it has already run for the same path.
func WithForceRun() PathOption {
	return func(pf *pathFilter) {
		pf.forceRun = true
	}
}

// WithDetect uses a detection function to discover paths dynamically.
// The detection function receives the pre-walked directory list and returns
// directories that match its criteria (e.g., directories containing go.mod).
//
// Inner detection functions resolve their paths against the current scope's
// candidates, allowing for refined path discovery.
//
// Example:
//
//	pk.WithOptions(
//	    golang.Tasks(),
//	    pk.WithExcludePath("vendor"), // Excludes vendor/ from the Go task scope
//	    pk.WithFlag(golang.Test, "race", true),
//	)
func WithDetect(fn DetectFunc) PathOption {
	return func(pf *pathFilter) {
		pf.detectFunc = fn
	}
}

// WithContextValue adds a key-value pair to the context for tasks in this scope.
//
// This is a low-level helper primarily for task package authors to pass runtime
// configuration to their tasks. End users should typically use pk.WithFlag
// instead of calling this directly.
func WithContextValue(key, value any) PathOption {
	return func(pf *pathFilter) {
		pf.contextValues = append(pf.contextValues, contextValue{key: key, value: value})
	}
}

// WithName adds a suffix to task names within this scope.
// The suffix is appended with a colon separator (e.g., "py-test" becomes "py-test:3.9").
// This affects CLI invocation, help output, and matrix generation.
//
// Example:
//
//	pk.WithOptions(
//	    pk.Parallel(taskA, taskB),
//	    pk.WithName("variant-1"),
//	)
func WithName(suffix string) PathOption {
	return func(pf *pathFilter) {
		pf.nameSuffix = suffix
	}
}

// WithOptions wraps a Runnable with path filtering options.
// The wrapped Runnable will execute in directories determined by
// include/exclude patterns resolved against the filesystem.
func WithOptions(r Runnable, opts ...PathOption) Runnable {
	pf := &pathFilter{
		inner:        r,
		includePaths: []string{},
		excludePaths: []excludePattern{},
		skippedTasks: []string{},
		flags:        []flagOverride{},
	}

	for _, opt := range opts {
		opt(pf)
	}

	return pf
}

// DetectByFile returns a DetectFunc that finds directories containing any of the specified files.
// For example, DetectByFile("go.mod") finds all Go modules.
func DetectByFile(filenames ...string) DetectFunc {
	return func(dirs []string, gitRoot string) []string {
		var result []string
		for _, dir := range dirs {
			absDir := filepath.Join(gitRoot, dir)
			for _, filename := range filenames {
				path := filepath.Join(absDir, filename)
				if _, err := os.Stat(path); err == nil {
					result = append(result, dir)
					break // Found a match, no need to check other filenames
				}
			}
		}
		return result
	}
}

// pathFilter wraps a Runnable with directory-based filtering.
// It determines which directories to execute in based on include/exclude patterns
// and optional detection functions.
type pathFilter struct {
	inner         Runnable
	includePaths  []string
	excludePaths  []excludePattern
	skippedTasks  []string
	flags         []flagOverride
	contextValues []contextValue // Key-value pairs to add to context.
	nameSuffix    string         // Suffix to append to task names (e.g., ":3.9").
	detectFunc    DetectFunc     // Optional detection function for dynamic path discovery.
	resolvedPaths []string       // Cached resolved paths from plan building.
	forceRun      bool           // Disable task deduplication for the wrapped Runnable.
}

type contextValue struct {
	key   any
	value any
}

type excludePattern struct {
	pattern string
	tasks   []string
}

type flagOverride struct {
	taskName string
	flagName string
	value    any
}

// run implements the Runnable interface.
// It executes the inner Runnable for each resolved path.
// Paths are resolved during plan building and cached in resolvedPaths.
// Flag overrides are pre-computed during planning and read from Plan.taskInstanceByName().
func (pf *pathFilter) run(ctx context.Context) error {
	// If forceRun is set, propagate it to the context.
	if pf.forceRun {
		ctx = withForceRun(ctx)
	}

	// Apply context values.
	for _, cv := range pf.contextValues {
		ctx = context.WithValue(ctx, cv.key, cv.value)
	}

	// Apply name suffix to context.
	if pf.nameSuffix != "" {
		ctx = withNameSuffix(ctx, pf.nameSuffix)
	}

	// Execute inner Runnable for each resolved path.
	for _, path := range pf.resolvedPaths {
		pathCtx := WithPath(ctx, path)
		if err := pf.inner.run(pathCtx); err != nil {
			return err
		}
	}
	return nil
}

func toTaskNames(tasks []any) []string {
	names := make([]string, 0, len(tasks))
	for _, t := range tasks {
		if name := toTaskName(t); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func toTaskName(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case interface{ Name() string }:
		return t.Name()
	default:
		return ""
	}
}
