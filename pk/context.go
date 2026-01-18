package pk

import "context"

// contextKey is the type for context keys in this package
type contextKey int

const (
	// pathKey is the context key for the current execution path
	pathKey contextKey = iota
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
