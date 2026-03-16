package pk

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk/internal/engine"
)

// PathFromContext returns the current execution path from the context.
// Returns "." if no path is set (meaning git root).
func PathFromContext(ctx context.Context) string {
	return engine.PathFromContext(ctx)
}

// Verbose returns whether verbose mode is enabled in the context.
func Verbose(ctx context.Context) bool {
	return engine.Verbose(ctx)
}

// ContextWithPath returns a new context with the given execution path.
// The path affects where Exec runs commands. Relative paths are resolved
// from the git root; absolute paths are used as-is.
//
//	ctx = pk.ContextWithPath(ctx, "services/api")
//	ctx = pk.ContextWithPath(ctx, repopath.FromPocketDir("tools", "mytool", "v1.0"))
func ContextWithPath(ctx context.Context, path string) context.Context {
	return engine.ContextWithPath(ctx, path)
}

// EnvConfig holds environment variable overrides applied to [Exec] calls.
// Built up via [ContextWithEnv] and [ContextWithoutEnv].
type EnvConfig = engine.EnvConfig

// ContextWithEnv returns a new context that sets an environment variable for Exec calls.
// The keyValue must be in "KEY=value" format.
// If a variable with the same key already exists, it is replaced.
//
//	ctx = pk.ContextWithEnv(ctx, "MY_VAR=value")
//	pk.Exec(ctx, "mycmd", "arg1") // runs with MY_VAR set
func ContextWithEnv(ctx context.Context, keyValue string) context.Context {
	return engine.ContextWithEnv(ctx, keyValue)
}

// ContextWithoutEnv returns a new context that filters out environment variables
// matching the given prefix from Exec calls.
//
//	ctx = pk.ContextWithoutEnv(ctx, "VIRTUAL_ENV")
//	pk.Exec(ctx, "python", "script.py") // runs without VIRTUAL_ENV
func ContextWithoutEnv(ctx context.Context, prefix string) context.Context {
	return engine.ContextWithoutEnv(ctx, prefix)
}

// EnvConfigFromContext returns a copy of the environment config from the context.
// Useful for inspecting what environment overrides are in effect.
func EnvConfigFromContext(ctx context.Context) EnvConfig {
	return engine.EnvConfigFromContext(ctx)
}
