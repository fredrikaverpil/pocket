package pk

import (
	"context"
	"maps"
	"slices"
	"strings"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Context Keys
// ═══════════════════════════════════════════════════════════════════════════════

type pathKey struct{}       // Current execution path.
type forceRunKey struct{}   // Forcing task execution.
type verboseKey struct{}    // Verbose mode.
type gitDiffKey struct{}    // Git diff enabled flag.
type envKey struct{}        // Environment variable overrides.
type nameSuffixKey struct{} // Task name suffix.
type autoExecKey struct{}   // Auto execution mode (manual tasks are skipped).

// ═══════════════════════════════════════════════════════════════════════════════
// Context Accessors (Getters)
// ═══════════════════════════════════════════════════════════════════════════════

// PathFromContext returns the current execution path from the context.
// Returns "." if no path is set (meaning git root).
func PathFromContext(ctx context.Context) string {
	if path, ok := ctx.Value(pathKey{}).(string); ok {
		return path
	}
	return "."
}

// Verbose returns whether verbose mode is enabled in the context.
func Verbose(ctx context.Context) bool {
	if v, ok := ctx.Value(verboseKey{}).(bool); ok {
		return v
	}
	return false
}

// gitDiffEnabledFromContext returns whether git diff is enabled in the context.
func gitDiffEnabledFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(gitDiffKey{}).(bool); ok {
		return v
	}
	return false
}

// isAutoExec returns whether auto execution mode is active.
func isAutoExec(ctx context.Context) bool {
	v, _ := ctx.Value(autoExecKey{}).(bool)
	return v
}

// nameSuffixFromContext returns the name suffix from the context.
// Returns empty string if no suffix is set.
func nameSuffixFromContext(ctx context.Context) string {
	if s, ok := ctx.Value(nameSuffixKey{}).(string); ok {
		return s
	}
	return ""
}

// forceRunFromContext returns whether forceRun is set in the context.
func forceRunFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(forceRunKey{}).(bool); ok {
		return v
	}
	return false
}

// ═══════════════════════════════════════════════════════════════════════════════
// Context Modifiers (Setters)
// ═══════════════════════════════════════════════════════════════════════════════

// ContextWithPath returns a new context with the given execution path.
// The path is relative to the git root and affects where Exec runs commands.
//
//	ctx = pk.ContextWithPath(ctx, "services/api")
//	pk.Exec(ctx, "go", "test", "./...") // runs in services/api/
func ContextWithPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, pathKey{}, path)
}

// contextWithVerbose returns a new context with verbose mode set.
func contextWithVerbose(ctx context.Context, verbose bool) context.Context {
	return context.WithValue(ctx, verboseKey{}, verbose)
}

// contextWithGitDiffEnabled returns a new context with git diff enabled flag set.
func contextWithGitDiffEnabled(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, gitDiffKey{}, enabled)
}

// contextWithAutoExec returns a new context with auto execution mode enabled.
// When auto exec is active, manual tasks are skipped.
func contextWithAutoExec(ctx context.Context) context.Context {
	return context.WithValue(ctx, autoExecKey{}, true)
}

// withForceRun returns a new context with forceRun set to true.
func withForceRun(ctx context.Context) context.Context {
	return context.WithValue(ctx, forceRunKey{}, true)
}

// contextWithNameSuffix returns a new context with the given name suffix.
// Name suffixes are accumulated (e.g., nested calls combine).
func contextWithNameSuffix(ctx context.Context, suffix string) context.Context {
	existing := nameSuffixFromContext(ctx)
	if existing != "" {
		suffix = existing + ":" + suffix
	}
	return context.WithValue(ctx, nameSuffixKey{}, suffix)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Environment Configuration
// ═══════════════════════════════════════════════════════════════════════════════

// EnvConfig holds environment variable overrides for command execution.
type EnvConfig struct {
	Set    map[string]string // key -> value (replaces existing)
	Filter []string          // prefixes to filter out
}

// ContextWithEnv returns a new context that sets an environment variable for Exec calls.
// The keyValue must be in "KEY=value" format.
// If a variable with the same key already exists, it is replaced.
//
//	ctx = pk.ContextWithEnv(ctx, "MY_VAR=value")
//	pk.Exec(ctx, "mycmd", "arg1") // runs with MY_VAR set
func ContextWithEnv(ctx context.Context, keyValue string) context.Context {
	key, value, ok := strings.Cut(keyValue, "=")
	if !ok {
		return ctx // invalid format, ignore
	}
	cfg := EnvConfigFromContext(ctx)
	if cfg.Set == nil {
		cfg.Set = make(map[string]string)
	}
	cfg.Set[key] = value
	return context.WithValue(ctx, envKey{}, cfg)
}

// ContextWithoutEnv returns a new context that filters out environment variables
// matching the given prefix from Exec calls.
//
//	ctx = pk.ContextWithoutEnv(ctx, "VIRTUAL_ENV")
//	pk.Exec(ctx, "python", "script.py") // runs without VIRTUAL_ENV
func ContextWithoutEnv(ctx context.Context, prefix string) context.Context {
	cfg := EnvConfigFromContext(ctx)
	cfg.Filter = append(cfg.Filter, prefix)
	return context.WithValue(ctx, envKey{}, cfg)
}

// EnvConfigFromContext returns the environment config from the context.
// Returns a copy to avoid mutating the original.
func EnvConfigFromContext(ctx context.Context) EnvConfig {
	cfg, ok := ctx.Value(envKey{}).(EnvConfig)
	if !ok {
		return EnvConfig{}
	}
	return EnvConfig{
		Set:    maps.Clone(cfg.Set),
		Filter: slices.Clone(cfg.Filter),
	}
}
