package run

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/fredrikaverpil/pocket/pk/internal/ctxkey"
)

// PathFromContext returns the current execution path from the context.
// Returns "." if no path is set (meaning git root).
func PathFromContext(ctx context.Context) string {
	if path, ok := ctx.Value(ctxkey.Path{}).(string); ok {
		return path
	}
	return "."
}

// Verbose returns whether verbose mode is enabled in the context.
func Verbose(ctx context.Context) bool {
	if v, ok := ctx.Value(ctxkey.Verbose{}).(bool); ok {
		return v
	}
	return false
}

// ContextWithPath returns a new context with the given execution path.
func ContextWithPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, ctxkey.Path{}, path)
}

// EnvConfig holds environment variable overrides applied to [Exec] calls.
type EnvConfig struct {
	// Set contains environment variables to set (or replace). Keys are variable names.
	Set map[string]string
	// Filter contains prefixes of environment variables to remove.
	Filter []string
}

// ContextWithEnv returns a new context that sets an environment variable
// for [Exec] calls. The keyValue must be in "KEY=value" format.
// Panics if keyValue does not contain "=".
func ContextWithEnv(ctx context.Context, keyValue string) context.Context {
	key, value, ok := strings.Cut(keyValue, "=")
	if !ok {
		panic("run.ContextWithEnv: keyValue must be in \"KEY=value\" format, got " + keyValue)
	}
	cfg := EnvConfigFromContext(ctx)
	if cfg.Set == nil {
		cfg.Set = make(map[string]string)
	}
	cfg.Set[key] = value
	return context.WithValue(ctx, ctxkey.Env{}, cfg)
}

// ContextWithoutEnv returns a new context that filters out environment
// variables matching the given prefix from [Exec] calls.
func ContextWithoutEnv(ctx context.Context, prefix string) context.Context {
	cfg := EnvConfigFromContext(ctx)
	cfg.Filter = append(cfg.Filter, prefix)
	return context.WithValue(ctx, ctxkey.Env{}, cfg)
}

// EnvConfigFromContext returns a copy of the environment config from the context.
func EnvConfigFromContext(ctx context.Context) EnvConfig {
	cfg, ok := ctx.Value(ctxkey.Env{}).(EnvConfig)
	if !ok {
		return EnvConfig{}
	}
	return EnvConfig{
		Set:    maps.Clone(cfg.Set),
		Filter: slices.Clone(cfg.Filter),
	}
}

// OutputFromContext returns the Output from the context.
// Returns nil if no output is set (caller should use [StdOutput]).
func OutputFromContext(ctx context.Context) *Output {
	if out, ok := ctx.Value(ctxkey.Output{}).(*Output); ok {
		return out
	}
	return nil
}

// outputOrStd returns the Output from context, falling back to StdOutput.
func outputOrStd(ctx context.Context) *Output {
	if out := OutputFromContext(ctx); out != nil {
		return out
	}
	return StdOutput()
}

// noticePatternsFromContext returns the notice patterns from context.
// Returns nil if not set (caller should use DefaultNoticePatterns).
func noticePatternsFromContext(ctx context.Context) []string {
	if patterns, ok := ctx.Value(ctxkey.NoticePatterns{}).([]string); ok {
		return patterns
	}
	return nil
}

// trackerFromContext returns the tracker from the context as any.
// Returns nil if no tracker is set.
func trackerFromContext(ctx context.Context) any {
	return ctx.Value(ctxkey.Tracker{})
}

// taskFlagsFromContext retrieves resolved flag values from context.
func taskFlagsFromContext(ctx context.Context) map[string]any {
	if flags, ok := ctx.Value(ctxkey.TaskFlags{}).(map[string]any); ok {
		return flags
	}
	return nil
}
