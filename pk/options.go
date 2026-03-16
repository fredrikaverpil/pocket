package pk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/fredrikaverpil/pocket/pk/internal/engine"
)

// Option configures execution behavior for a Runnable within a [WithOptions] scope.
// Options control path filtering, flag overrides, task skipping, deduplication,
// and other scope-level settings.
type Option func(*pathFilter)

// DetectFunc is a function that filters directories to find those with specific markers.
// It receives the pre-walked directory list and git root path, and returns matching directories.
// Used with WithDetect to dynamically discover paths for task execution.
type DetectFunc func(dirs []string, gitRoot string) []string

// WithPath adds include patterns for path filtering.
// Only directories matching any of the patterns will be included.
// Patterns are relative to the git root and are interpreted as regular expressions.
func WithPath(patterns ...string) Option {
	return func(pf *pathFilter) {
		pf.includePaths = append(pf.includePaths, patterns...)
	}
}

// WithSkipPath adds exclude patterns for path filtering.
// Directories matching any of the patterns will be excluded for ALL tasks in the current scope.
// Patterns are relative to the git root and are interpreted as regular expressions.
func WithSkipPath(patterns ...string) Option {
	return func(pf *pathFilter) {
		for _, p := range patterns {
			pf.excludePaths = append(pf.excludePaths, excludePattern{
				pattern: p,
				tasks:   nil, // Global for the scope.
			})
		}
	}
}

// WithSkipTask skips a task within the current scope.
// When called with only a task, the task is removed entirely from the scope.
// When called with a task and patterns, the task is excluded from directories matching the patterns.
// Patterns are relative to the git root and are interpreted as regular expressions.
// The task can be specified by its string name or by the task object itself.
func WithSkipTask(task any, patterns ...string) Option {
	return func(pf *pathFilter) {
		name := toTaskName(task)
		if len(patterns) == 0 {
			pf.skippedTasks = append(pf.skippedTasks, name)
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

// WithFlags sets flag overrides for a task in the current scope.
// The task is inferred by matching the flags struct type against
// the Flags field of tasks in scope. The flags struct must be the
// same type as exactly one task's Flags field within the scope.
// Only fields that differ from the task's defaults are applied as overrides.
func WithFlags(flags any) Option {
	return func(pf *pathFilter) {
		ft := reflect.TypeOf(flags)
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		pf.flags = append(pf.flags, flagOverride{
			flagsType: ft,
			flags:     flags,
		})
	}
}

// WithForceRun disables task deduplication for the wrapped Runnable.
// By default, tasks are deduplicated per (task pointer, path) pair within
// a single invocation. WithForceRun causes the task to always execute,
// even if it has already run for the same path.
func WithForceRun() Option {
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
//	    pk.WithSkipPath("vendor"), // Excludes vendor/ from the Go task scope
//	    pk.WithFlags(golang.TestFlags{Race: true}),
//	)
func WithDetect(fn DetectFunc) Option {
	return func(pf *pathFilter) {
		pf.detectFunc = fn
	}
}

// WithNoticePatterns sets the patterns used to detect warning-like output
// from commands in the current scope. Uses replace semantics (not append).
// Pass no patterns to disable notice detection entirely.
func WithNoticePatterns(patterns ...string) Option {
	return func(pf *pathFilter) {
		pf.noticePatterns = patterns
	}
}

// WithNameSuffix creates a named variant of tasks within this scope.
// The suffix is appended with a colon separator (e.g., "py-test" becomes "py-test:3.9").
//
// Use this to create distinct task instances from the same task definition.
// Each variant is deduplicated separately, so "py-test:3.9" and "py-test:3.10"
// both run even though they share the same underlying task.
//
// Example:
//
//	pk.WithOptions(python.Test, pk.WithNameSuffix("3.9"), pk.WithFlags(python.Flags{Python: "3.9"}))
//	pk.WithOptions(python.Test, pk.WithNameSuffix("3.10"), pk.WithFlags(python.Flags{Python: "3.10"}))
func WithNameSuffix(suffix string) Option {
	return func(pf *pathFilter) {
		pf.nameSuffix = suffix
	}
}

// WithOptions wraps a Runnable with path filtering options.
// The wrapped Runnable will execute in directories determined by
// include/exclude patterns resolved against the filesystem.
func WithOptions(r Runnable, opts ...Option) Runnable {
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
	inner          Runnable
	includePaths   []string
	excludePaths   []excludePattern
	skippedTasks   []string
	flags          []flagOverride
	nameSuffix     string     // Suffix to append to task names (e.g., ":3.9").
	detectFunc     DetectFunc // Optional detection function for dynamic path discovery.
	resolvedPaths  []string   // Cached resolved paths from plan building.
	forceRun       bool       // Disable task deduplication for the wrapped Runnable.
	noticePatterns []string   // Custom notice detection patterns (nil = use default).
}

type excludePattern struct {
	pattern string
	tasks   []string
}

type flagOverride struct {
	taskName  string       // Set when task is known (resolved or explicit).
	flagName  string       // Individual flag name.
	value     any          // Individual flag value.
	flagsType reflect.Type // Set when task should be inferred from flags type.
	flags     any          // The full flags struct (for deferred resolution).
}

// run implements the Runnable interface.
// It executes the inner Runnable for each resolved path.
// Paths are resolved during plan building and cached in resolvedPaths.
// Flag overrides are pre-computed during planning and read from Plan.taskInstanceByName().
func (pf *pathFilter) run(ctx context.Context) error {
	// If forceRun is set, propagate it to the context.
	if pf.forceRun {
		ctx = engine.WithForceRun(ctx)
	}

	// Apply name suffix to context.
	if pf.nameSuffix != "" {
		ctx = engine.ContextWithNameSuffix(ctx, pf.nameSuffix)
	}

	// Apply notice patterns to context (nil means use default, empty slice disables).
	if pf.noticePatterns != nil {
		ctx = engine.SetNoticePatterns(ctx, pf.noticePatterns)
	}

	// Execute inner Runnable for each resolved path.
	for _, path := range pf.resolvedPaths {
		pathCtx := ContextWithPath(ctx, path)
		if err := pf.inner.run(pathCtx); err != nil {
			return err
		}
	}
	return nil
}

// resolveTypedFlags resolves flagOverrides that use flagsType (deferred resolution)
// by matching against tasks found in the inner runnable. Returns resolved flagOverrides
// with taskName filled in.
func resolveTypedFlags(flags []flagOverride, inner Runnable) ([]flagOverride, error) {
	resolved := make([]flagOverride, 0, len(flags))
	for _, f := range flags {
		if f.flagsType == nil {
			// Already resolved (has taskName + individual flag entries).
			resolved = append(resolved, f)
			continue
		}

		// Find task with matching Flags type.
		taskName, task, err := findTaskByFlagsType(inner, f.flagsType)
		if err != nil {
			return nil, err
		}

		// Compute diff and expand to individual flag overrides.
		diff, err := engine.DiffStructs(task.Flags, f.flags)
		if err != nil {
			return nil, fmt.Errorf("pk.WithFlags: %v", err)
		}
		for name, value := range diff {
			resolved = append(resolved, flagOverride{
				taskName: taskName,
				flagName: name,
				value:    value,
			})
		}
	}
	return resolved, nil
}

// findTaskByFlagsType walks a Runnable tree to find the task whose Flags field
// matches the given type. Returns an error if zero or multiple tasks match.
func findTaskByFlagsType(r Runnable, ft reflect.Type) (string, *Task, error) {
	var matches []*Task
	walkTasks(r, func(t *Task) {
		if t.Flags == nil {
			return
		}
		tt := reflect.TypeOf(t.Flags)
		if tt.Kind() == reflect.Pointer {
			tt = tt.Elem()
		}
		if tt == ft {
			matches = append(matches, t)
		}
	})

	switch len(matches) {
	case 0:
		return "", nil, fmt.Errorf("pk.WithFlags: no task found with flags type %v", ft)
	case 1:
		return matches[0].Name, matches[0], nil
	default:
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = m.Name
		}
		return "", nil, fmt.Errorf("pk.WithFlags: ambiguous flags type %v matches tasks: %v", ft, names)
	}
}

// walkTasks calls fn for every *Task reachable from r.
func walkTasks(r Runnable, fn func(*Task)) {
	switch v := r.(type) {
	case *Task:
		fn(v)
	case *serial:
		for _, child := range v.runnables {
			walkTasks(child, fn)
		}
	case *parallel:
		for _, child := range v.runnables {
			walkTasks(child, fn)
		}
	case *pathFilter:
		walkTasks(v.inner, fn)
	}
}

func toTaskName(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case *Task:
		return t.Name
	default:
		panic(fmt.Sprintf("pk: unsupported task type %T (expected string or *pk.Task)", v))
	}
}
