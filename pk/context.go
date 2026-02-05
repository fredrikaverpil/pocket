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

// contextKey is the type for context keys in this package.
type contextKey int

const (
	// pathKey is the context key for the current execution path.
	pathKey contextKey = iota
	// forceRunKey is the context key for forcing task execution.
	forceRunKey
	// verboseKey is the context key for verbose mode.
	verboseKey
	// gitDiffKey is the context key for git diff enabled flag.
	gitDiffKey
	// envKey is the context key for environment variable overrides.
	envKey
	// nameSuffixKey is the context key for task name suffix.
	nameSuffixKey
	// autoExecKey is the context key for auto execution mode.
	// When set, manual tasks are skipped.
	autoExecKey
)

// ═══════════════════════════════════════════════════════════════════════════════
// Context Accessors (Getters)
// ═══════════════════════════════════════════════════════════════════════════════

// PathFromContext returns the current execution path from the context.
// Returns "." if no path is set (meaning git root).
func PathFromContext(ctx context.Context) string {
	if path, ok := ctx.Value(pathKey).(string); ok {
		return path
	}
	return "."
}

// Verbose returns whether verbose mode is enabled in the context.
func Verbose(ctx context.Context) bool {
	if v, ok := ctx.Value(verboseKey).(bool); ok {
		return v
	}
	return false
}

// GitDiffEnabledFromContext returns whether git diff is enabled in the context.
func GitDiffEnabledFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(gitDiffKey).(bool); ok {
		return v
	}
	return false
}

// IsAutoExec returns whether auto execution mode is active.
func IsAutoExec(ctx context.Context) bool {
	v, _ := ctx.Value(autoExecKey).(bool)
	return v
}

// NameSuffixFromContext returns the name suffix from the context.
// Returns empty string if no suffix is set.
func NameSuffixFromContext(ctx context.Context) string {
	if s, ok := ctx.Value(nameSuffixKey).(string); ok {
		return s
	}
	return ""
}

// ForceRunFromContext returns whether forceRun is set in the context.
func ForceRunFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(forceRunKey).(bool); ok {
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
	return context.WithValue(ctx, pathKey, path)
}

// ContextWithVerbose returns a new context with verbose mode set.
func ContextWithVerbose(ctx context.Context, verbose bool) context.Context {
	return context.WithValue(ctx, verboseKey, verbose)
}

// ContextWithGitDiffEnabled returns a new context with git diff enabled flag set.
func ContextWithGitDiffEnabled(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, gitDiffKey, enabled)
}

// ContextWithAutoExec returns a new context with auto execution mode enabled.
// When auto exec is active, manual tasks are skipped.
func ContextWithAutoExec(ctx context.Context) context.Context {
	return context.WithValue(ctx, autoExecKey, true)
}

// ContextWithNameSuffix returns a new context with the given name suffix.
// Name suffixes are accumulated (e.g., nested calls combine).
func ContextWithNameSuffix(ctx context.Context, suffix string) context.Context {
	existing := NameSuffixFromContext(ctx)
	if existing != "" {
		suffix = existing + ":" + suffix
	}
	return context.WithValue(ctx, nameSuffixKey, suffix)
}

// ContextWithForceRun returns a new context with forceRun set to true.
func ContextWithForceRun(ctx context.Context) context.Context {
	return context.WithValue(ctx, forceRunKey, true)
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
	return context.WithValue(ctx, envKey, cfg)
}

// ContextWithoutEnv returns a new context that filters out environment variables
// matching the given prefix from Exec calls.
//
//	ctx = pk.ContextWithoutEnv(ctx, "VIRTUAL_ENV")
//	pk.Exec(ctx, "python", "script.py") // runs without VIRTUAL_ENV
func ContextWithoutEnv(ctx context.Context, prefix string) context.Context {
	cfg := EnvConfigFromContext(ctx)
	cfg.Filter = append(cfg.Filter, prefix)
	return context.WithValue(ctx, envKey, cfg)
}

// EnvConfigFromContext returns the environment config from the context.
// Returns a copy to avoid mutating the original.
func EnvConfigFromContext(ctx context.Context) EnvConfig {
	cfg, ok := ctx.Value(envKey).(EnvConfig)
	if !ok {
		return EnvConfig{}
	}
	return EnvConfig{
		Set:    maps.Clone(cfg.Set),
		Filter: slices.Clone(cfg.Filter),
	}
}
