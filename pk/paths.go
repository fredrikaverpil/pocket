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
	inner        Runnable
	includePaths []string
	excludePaths []string
	detect       func() []string
}

// run implements the Runnable interface.
// It resolves paths and executes the inner Runnable for each path.
func (pf *pathFilter) run(ctx context.Context) error {
	// TODO: Implement path resolution
	// For now, just run the inner Runnable once
	return pf.inner.run(ctx)
}
