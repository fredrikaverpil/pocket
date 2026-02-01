package pk

import (
	"context"
	"maps"
	"slices"
	"strings"
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

// withGitDiffEnabled returns a new context with git diff enabled flag set.
func withGitDiffEnabled(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, gitDiffKey, enabled)
}

// gitDiffEnabledFromContext returns whether git diff is enabled in the context.
func gitDiffEnabledFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(gitDiffKey).(bool); ok {
		return v
	}
	return false
}

// withAutoExec returns a new context with auto execution mode enabled.
// When auto exec is active, manual tasks are skipped.
func withAutoExec(ctx context.Context) context.Context {
	return context.WithValue(ctx, autoExecKey, true)
}

// isAutoExec returns whether auto execution mode is active.
func isAutoExec(ctx context.Context) bool {
	v, _ := ctx.Value(autoExecKey).(bool)
	return v
}

// envConfig holds environment variable overrides for command execution.
type envConfig struct {
	set    map[string]string // key -> value (replaces existing)
	filter []string          // prefixes to filter out
}

// WithEnv returns a new context that sets an environment variable.
// The keyValue must be in "KEY=value" format.
// If a variable with the same key already exists, it is replaced.
func WithEnv(ctx context.Context, keyValue string) context.Context {
	key, value, ok := strings.Cut(keyValue, "=")
	if !ok {
		return ctx // invalid format, ignore
	}
	cfg := envConfigFromContext(ctx)
	if cfg.set == nil {
		cfg.set = make(map[string]string)
	}
	cfg.set[key] = value
	return context.WithValue(ctx, envKey, cfg)
}

// WithoutEnv returns a new context that filters out environment variables
// matching the given prefix. For example, WithoutEnv(ctx, "VIRTUAL_ENV")
// removes VIRTUAL_ENV from the environment.
func WithoutEnv(ctx context.Context, prefix string) context.Context {
	cfg := envConfigFromContext(ctx)
	cfg.filter = append(cfg.filter, prefix)
	return context.WithValue(ctx, envKey, cfg)
}

// envConfigFromContext returns the environment config from the context.
// Returns a copy to avoid mutating the original.
func envConfigFromContext(ctx context.Context) envConfig {
	cfg, ok := ctx.Value(envKey).(envConfig)
	if !ok {
		return envConfig{}
	}
	return envConfig{
		set:    maps.Clone(cfg.set),
		filter: slices.Clone(cfg.filter),
	}
}

// withNameSuffix returns a new context with the given name suffix.
// Name suffixes are accumulated (e.g., nested WithNameSuffix calls combine).
func withNameSuffix(ctx context.Context, suffix string) context.Context {
	existing := nameSuffixFromContext(ctx)
	if existing != "" {
		suffix = existing + ":" + suffix
	}
	return context.WithValue(ctx, nameSuffixKey, suffix)
}

// nameSuffixFromContext returns the name suffix from the context.
// Returns empty string if no suffix is set.
func nameSuffixFromContext(ctx context.Context) string {
	if s, ok := ctx.Value(nameSuffixKey).(string); ok {
		return s
	}
	return ""
}
