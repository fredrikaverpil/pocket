package run

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/internal/engine"
)

// Exec runs an external command with .pocket/bin prepended to PATH.
// See [engine.Exec] for full documentation.
func Exec(ctx context.Context, name string, args ...string) error {
	return engine.Exec(ctx, name, args...)
}

// Printf formats and writes to the context's stdout.
func Printf(ctx context.Context, format string, a ...any) {
	engine.Printf(ctx, format, a...)
}

// Println writes to the context's stdout, appending a newline.
func Println(ctx context.Context, a ...any) {
	engine.Println(ctx, a...)
}

// Errorf formats and writes to the context's stderr.
func Errorf(ctx context.Context, format string, a ...any) {
	engine.Errorf(ctx, format, a...)
}

// GetFlags retrieves the resolved flags for a task from context.
func GetFlags[T any](ctx context.Context) T {
	return engine.GetFlags[T](ctx)
}

// PathFromContext returns the current execution path from the context.
func PathFromContext(ctx context.Context) string {
	return engine.PathFromContext(ctx)
}

// Verbose returns whether verbose mode is enabled in the context.
func Verbose(ctx context.Context) bool {
	return engine.Verbose(ctx)
}

// ContextWithPath returns a new context with the given execution path.
func ContextWithPath(ctx context.Context, path string) context.Context {
	return engine.ContextWithPath(ctx, path)
}

// ContextWithEnv returns a new context that sets an environment variable
// for [Exec] calls. The keyValue must be in "KEY=value" format.
func ContextWithEnv(ctx context.Context, keyValue string) context.Context {
	return engine.ContextWithEnv(ctx, keyValue)
}

// ContextWithoutEnv returns a new context that filters out environment
// variables matching the given prefix from [Exec] calls.
func ContextWithoutEnv(ctx context.Context, prefix string) context.Context {
	return engine.ContextWithoutEnv(ctx, prefix)
}

// EnvConfig holds environment variable overrides applied to [Exec] calls.
type EnvConfig = engine.EnvConfig

// EnvConfigFromContext returns a copy of the environment config from the context.
func EnvConfigFromContext(ctx context.Context) EnvConfig {
	return engine.EnvConfigFromContext(ctx)
}

// PlanFromContext returns the Plan from the context.
// Returns nil if no plan is set.
func PlanFromContext(ctx context.Context) *pk.Plan {
	v := engine.PlanFromContext(ctx)
	if v == nil {
		return nil
	}
	return v.(*pk.Plan)
}

// RegisterPATH registers a directory to be added to PATH for all [Exec] calls.
func RegisterPATH(dir string) {
	engine.RegisterPATH(dir)
}

// DefaultNoticePatterns are the substrings used to detect warning-like output.
var DefaultNoticePatterns = engine.DefaultNoticePatterns
