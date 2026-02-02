// Package ctx provides context management for Pocket task execution.
// It contains context keys and accessors for passing values through the task graph.
//
// This package is a leaf package with no dependencies on other pk packages,
// allowing it to be imported by both pk/ and internal/core/ without cycles.
package pcontext

import (
	"context"
	"maps"
	"slices"
	"strings"
)

// ContextKey is the type for context keys in this package.
// Exported so other packages can define their own context accessors using these keys.
type ContextKey int

const (
	// PathKey is the context key for the current execution path.
	PathKey ContextKey = iota
	// ForceRunKey is the context key for forcing task execution.
	ForceRunKey
	// VerboseKey is the context key for verbose mode.
	VerboseKey
	// GitDiffKey is the context key for git diff enabled flag.
	GitDiffKey
	// EnvKey is the context key for environment variable overrides.
	EnvKey
	// NameSuffixKey is the context key for task name suffix.
	NameSuffixKey
	// AutoExecKey is the context key for auto execution mode.
	// When set, manual tasks are skipped.
	AutoExecKey
)

// WithPath returns a new context with the given path set.
// The path is relative to the git root.
func WithPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, PathKey, path)
}

// PathFromContext returns the current execution path from the context.
// Returns "." if no path is set (meaning git root).
func PathFromContext(ctx context.Context) string {
	if path, ok := ctx.Value(PathKey).(string); ok {
		return path
	}
	return "."
}

// WithVerbose returns a new context with verbose mode set.
func WithVerbose(ctx context.Context, verbose bool) context.Context {
	return context.WithValue(ctx, VerboseKey, verbose)
}

// Verbose returns whether verbose mode is enabled in the context.
func Verbose(ctx context.Context) bool {
	if v, ok := ctx.Value(VerboseKey).(bool); ok {
		return v
	}
	return false
}

// WithGitDiffEnabled returns a new context with git diff enabled flag set.
func WithGitDiffEnabled(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, GitDiffKey, enabled)
}

// GitDiffEnabledFromContext returns whether git diff is enabled in the context.
func GitDiffEnabledFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(GitDiffKey).(bool); ok {
		return v
	}
	return false
}

// WithAutoExec returns a new context with auto execution mode enabled.
// When auto exec is active, manual tasks are skipped.
func WithAutoExec(ctx context.Context) context.Context {
	return context.WithValue(ctx, AutoExecKey, true)
}

// IsAutoExec returns whether auto execution mode is active.
func IsAutoExec(ctx context.Context) bool {
	v, _ := ctx.Value(AutoExecKey).(bool)
	return v
}

// EnvConfig holds environment variable overrides for command execution.
type EnvConfig struct {
	Set    map[string]string // key -> value (replaces existing)
	Filter []string          // prefixes to filter out
}

// WithEnv returns a new context that sets an environment variable.
// The keyValue must be in "KEY=value" format.
// If a variable with the same key already exists, it is replaced.
func WithEnv(ctx context.Context, keyValue string) context.Context {
	key, value, ok := strings.Cut(keyValue, "=")
	if !ok {
		return ctx // invalid format, ignore
	}
	cfg := EnvConfigFromContext(ctx)
	if cfg.Set == nil {
		cfg.Set = make(map[string]string)
	}
	cfg.Set[key] = value
	return context.WithValue(ctx, EnvKey, cfg)
}

// WithoutEnv returns a new context that filters out environment variables
// matching the given prefix. For example, WithoutEnv(ctx, "VIRTUAL_ENV")
// removes VIRTUAL_ENV from the environment.
func WithoutEnv(ctx context.Context, prefix string) context.Context {
	cfg := EnvConfigFromContext(ctx)
	cfg.Filter = append(cfg.Filter, prefix)
	return context.WithValue(ctx, EnvKey, cfg)
}

// EnvConfigFromContext returns the environment config from the context.
// Returns a copy to avoid mutating the original.
func EnvConfigFromContext(ctx context.Context) EnvConfig {
	cfg, ok := ctx.Value(EnvKey).(EnvConfig)
	if !ok {
		return EnvConfig{}
	}
	return EnvConfig{
		Set:    maps.Clone(cfg.Set),
		Filter: slices.Clone(cfg.Filter),
	}
}

// WithNameSuffix returns a new context with the given name suffix.
// Name suffixes are accumulated (e.g., nested WithNameSuffix calls combine).
func WithNameSuffix(ctx context.Context, suffix string) context.Context {
	existing := NameSuffixFromContext(ctx)
	if existing != "" {
		suffix = existing + ":" + suffix
	}
	return context.WithValue(ctx, NameSuffixKey, suffix)
}

// NameSuffixFromContext returns the name suffix from the context.
// Returns empty string if no suffix is set.
func NameSuffixFromContext(ctx context.Context) string {
	if s, ok := ctx.Value(NameSuffixKey).(string); ok {
		return s
	}
	return ""
}

// WithForceRun returns a new context with forceRun set to true.
func WithForceRun(ctx context.Context) context.Context {
	return context.WithValue(ctx, ForceRunKey, true)
}

// ForceRunFromContext returns whether forceRun is set in the context.
func ForceRunFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(ForceRunKey).(bool); ok {
		return v
	}
	return false
}
