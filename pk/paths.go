package pk

import "context"

// PathOption configures path filtering for a Runnable.
// Path options determine which directories a task should execute in.
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
// It determines which directories to execute in based on include/exclude patterns.
type pathFilter struct {
	inner         Runnable
	includePaths  []string
	excludePaths  []string
	resolvedPaths []string // Cached resolved paths from plan building
}

// run implements the Runnable interface.
// It executes the inner Runnable for each resolved path.
// Paths are resolved during plan building and cached in resolvedPaths.
func (pf *pathFilter) run(ctx context.Context) error {
	// Execute inner Runnable for each resolved path
	for _, path := range pf.resolvedPaths {
		pathCtx := WithPath(ctx, path)
		if err := pf.inner.run(pathCtx); err != nil {
			return err
		}
	}
	return nil
}
