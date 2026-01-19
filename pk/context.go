package pk

import (
	"context"
	"sync"
)

// contextKey is the type for context keys in this package.
type contextKey int

const (
	// pathKey is the context key for the current execution path.
	pathKey contextKey = iota
	// planKey is the context key for the execution plan.
	planKey
	// trackerKey is the context key for the execution tracker.
	trackerKey
	// forceRunKey is the context key for forcing task execution.
	forceRunKey
)

// WithPath returns a new context with the given path set.
// The path is relative to the git root.
func WithPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, pathKey, path)
}

// PathFromContext returns the current execution path from the context.
// Returns "." if no path is set (meaning git root).
func PathFromContext(ctx context.Context) string {
	if path, ok := ctx.Value(pathKey).(string); ok {
		return path
	}
	return "."
}

// WithPlan returns a new context with the given plan set.
// This is used internally to pass the plan through execution.
func WithPlan(ctx context.Context, p *plan) context.Context {
	return context.WithValue(ctx, planKey, p)
}

// executionTracker tracks which tasks have already executed.
// It is safe for concurrent use. Deduplication is global by task pointer -
// the same *Task instance runs at most once per invocation, regardless of
// which path context it's executed in.
type executionTracker struct {
	mu   sync.Mutex
	done map[*Task]bool
}

// newExecutionTracker creates a new execution tracker.
func newExecutionTracker() *executionTracker {
	return &executionTracker{
		done: make(map[*Task]bool),
	}
}

// markDone records that a task has executed.
// Returns true if it was already done (should skip), false if first time.
func (t *executionTracker) markDone(task *Task) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done[task] {
		return true // already done, should skip
	}
	t.done[task] = true
	return false // first time
}

// withExecutionTracker returns a new context with the given tracker set.
func withExecutionTracker(ctx context.Context, t *executionTracker) context.Context {
	return context.WithValue(ctx, trackerKey, t)
}

// executionTrackerFromContext returns the execution tracker from the context.
// Returns nil if no tracker is set.
func executionTrackerFromContext(ctx context.Context) *executionTracker {
	if t, ok := ctx.Value(trackerKey).(*executionTracker); ok {
		return t
	}
	return nil
}

// withForceRun returns a new context with forceRun set to true.
func withForceRun(ctx context.Context) context.Context {
	return context.WithValue(ctx, forceRunKey, true)
}

// forceRunFromContext returns whether forceRun is set in the context.
func forceRunFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(forceRunKey).(bool); ok {
		return v
	}
	return false
}
