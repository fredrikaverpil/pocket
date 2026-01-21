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
	// verboseKey is the context key for verbose mode.
	verboseKey
	// outputKey is the context key for output writers.
	outputKey
	// flagsKey is the context key for task flag overrides.
	flagsKey
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

// WithPlan returns a new context with the given Plan set.
// This is used internally to pass the plan through execution.
func WithPlan(ctx context.Context, p *Plan) context.Context {
	return context.WithValue(ctx, planKey, p)
}

// PlanFromContext returns the Plan from the context.
// Returns nil if no plan is set.
func PlanFromContext(ctx context.Context) *Plan {
	if p, ok := ctx.Value(planKey).(*Plan); ok {
		return p
	}
	return nil
}

// executionTracker tracks which tasks have already executed.
// It is safe for concurrent use. Deduplication is by (task name, path) tuple -
// the same task can run multiple times if configured for different paths,
// but will only run once per path.
type executionTracker struct {
	mu   sync.Mutex
	done map[string]bool // key: "taskName:path"
}

// newExecutionTracker creates a new execution tracker.
func newExecutionTracker() *executionTracker {
	return &executionTracker{
		done: make(map[string]bool),
	}
}

// markDone records that a task has executed in a given path.
// Returns true if it was already done (should skip), false if first time.
func (t *executionTracker) markDone(taskName, path string) bool {
	key := taskName + ":" + path
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done[key] {
		return true // already done, should skip
	}
	t.done[key] = true
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

// WithVerbose returns a new context with verbose mode set.
func WithVerbose(ctx context.Context, verbose bool) context.Context {
	return context.WithValue(ctx, verboseKey, verbose)
}

// Verbose returns whether verbose mode is enabled in the context.
func Verbose(ctx context.Context) bool {
	if v, ok := ctx.Value(verboseKey).(bool); ok {
		return v
	}
	return false
}

// WithOutput returns a new context with the given output set.
func WithOutput(ctx context.Context, out *Output) context.Context {
	return context.WithValue(ctx, outputKey, out)
}

// withFlagOverride returns a new context with a flag override for a specific task.
func withFlagOverride(ctx context.Context, taskName, flagName string, value any) context.Context {
	overrides := flagOverridesFromContext(ctx)
	newOverrides := make(map[string]map[string]any)

	// Shallow copy outer map.
	for k, v := range overrides {
		newOverrides[k] = v
	}

	// Shallow copy inner map for the specific task.
	inner := make(map[string]any)
	for k, v := range overrides[taskName] {
		inner[k] = v
	}
	inner[flagName] = value
	newOverrides[taskName] = inner

	return context.WithValue(ctx, flagsKey, newOverrides)
}

// flagOverridesFromContext returns the map of task flag overrides from the context.
func flagOverridesFromContext(ctx context.Context) map[string]map[string]any {
	if v, ok := ctx.Value(flagsKey).(map[string]map[string]any); ok {
		return v
	}
	return nil
}

// OutputFromContext returns the Output from the context.
// Returns StdOutput() if no output is set.
func OutputFromContext(ctx context.Context) *Output {
	if out, ok := ctx.Value(outputKey).(*Output); ok {
		return out
	}
	return StdOutput()
}
