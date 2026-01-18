package pk

import (
	"context"
)

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

// Detect sets a detection function that returns candidate directories.
// The function should traverse the filesystem and return paths where
// the task should run (e.g., directories containing go.mod files).
//
// NOTE: This is currently stored but not used in path resolution.
// Reserved for future auto-detection functionality.
func Detect(fn func() []string) PathOption {
	return func(pf *pathFilter) {
		pf.detect = fn
	}
}

// WithOptions wraps a Runnable with path filtering options.
// The wrapped Runnable will execute in directories determined by
// the detection function and include/exclude patterns.
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
// It determines which directories to execute in based on detection
// functions and include/exclude patterns.
type pathFilter struct {
	inner         Runnable
	includePaths  []string
	excludePaths  []string
	detect        func() []string
	resolvedPaths []string // Cached resolved paths from plan building
}

// run implements the Runnable interface.
// It executes the inner Runnable for each resolved path.
// Paths are resolved during plan building and cached in resolvedPaths.
func (pf *pathFilter) run(ctx context.Context) error {
	// Use cached resolved paths from plan building
	resolvedPaths := pf.resolvedPaths

	// If no cached paths (shouldn't happen if plan was built), resolve now
	if len(resolvedPaths) == 0 {
		gitRoot := findGitRoot()
		var err error
		resolvedPaths, err = resolvePathPatterns(gitRoot, pf.includePaths, pf.excludePaths)
		if err != nil || len(resolvedPaths) == 0 {
			// Fall back to running at root
			resolvedPaths = []string{"."}
		}
	}

	// Execute inner Runnable for each resolved path
	for _, path := range resolvedPaths {
		// Set path in context
		pathCtx := WithPath(ctx, path)

		// Run inner Runnable with path context
		if err := pf.inner.run(pathCtx); err != nil {
			return err
		}
	}

	return nil
}
