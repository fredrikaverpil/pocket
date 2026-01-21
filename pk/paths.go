package pk

import "context"

// PathOption configures path filtering and execution behavior for a Runnable.
// Path options determine which directories a task should execute in and
// control deduplication behavior.
type PathOption func(*pathFilter)

// WithIncludePath adds an include pattern for path filtering.
// Only directories matching the pattern will be included.
// Patterns are relative to the git root.
func WithIncludePath(pattern string) PathOption {
	return func(pf *pathFilter) {
		pf.includePaths = append(pf.includePaths, pattern)
	}
}

// WithExcludePath adds an exclude pattern for path filtering.
// Directories matching the pattern will be excluded.
// Patterns are relative to the git root.
func WithExcludePath(pattern string) PathOption {
	return func(pf *pathFilter) {
		pf.excludePaths = append(pf.excludePaths, pattern)
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
// Combine with WithExcludePath to filter out specific directories.
//
// Example:
//
//	pk.WithOptions(
//	    golang.Tasks(),
//	    pk.WithDetect(pk.DetectByFile("go.mod")),
//	    pk.WithExcludePath("vendor"),
//	)
func WithDetect(fn DetectFunc) PathOption {
	return func(pf *pathFilter) {
		pf.detectFunc = fn
	}
}

// WithOptions wraps a Runnable with path filtering options.
// The wrapped Runnable will execute in directories determined by
// include/exclude patterns resolved against the filesystem.
func WithOptions(r Runnable, opts ...PathOption) Runnable {
	pf := &pathFilter{
		inner:        r,
		includePaths: []string{},
		excludePaths: []string{},
	}

	for _, opt := range opts {
		opt(pf)
	}

	return pf
}

// pathFilter wraps a Runnable with directory-based filtering.
// It determines which directories to execute in based on include/exclude patterns
// and optional detection functions.
type pathFilter struct {
	inner         Runnable
	includePaths  []string
	excludePaths  []string
	detectFunc    DetectFunc // Optional detection function for dynamic path discovery.
	resolvedPaths []string   // Cached resolved paths from plan building.
	forceRun      bool       // Disable task deduplication for the wrapped Runnable.
}

// run implements the Runnable interface.
// It executes the inner Runnable for each resolved path.
// Paths are resolved during plan building and cached in resolvedPaths.
func (pf *pathFilter) run(ctx context.Context) error {
	// If forceRun is set, propagate it to the context.
	if pf.forceRun {
		ctx = withForceRun(ctx)
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
