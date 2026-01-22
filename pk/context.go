package pk

import "context"

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

// OutputFromContext returns the Output from the context.
// Returns StdOutput() if no output is set.
func OutputFromContext(ctx context.Context) *Output {
	if out, ok := ctx.Value(outputKey).(*Output); ok {
		return out
	}
	return StdOutput()
}
