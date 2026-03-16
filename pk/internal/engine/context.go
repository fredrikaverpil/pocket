package engine

import (
	"context"
	"maps"
	"slices"
	"strings"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Context Keys
// ═══════════════════════════════════════════════════════════════════════════════

type (
	pathKey         struct{} // Current execution path.
	forceRunKey     struct{} // Forcing task execution.
	verboseKey      struct{} // Verbose mode.
	gitDiffKey      struct{} // Git diff enabled flag.
	commitsCheckKey struct{} // Commits check enabled flag.
	envKey          struct{} // Environment variable overrides.
	nameSuffixKey   struct{} // Task name suffix.
	autoExecKey     struct{} // Auto execution mode (manual tasks are skipped).
	taskFlagsKey    struct{} // Resolved task flag values.
	cliFlagsKey     struct{} // CLI-provided flag overrides.
	noticePatternsKey struct{} // Custom notice patterns.
	planKey         struct{} // Execution plan (stored as any).
	trackerKey      struct{} // Execution tracker (stored as any).
	outputKey       struct{} // Output writers.
)

// WarningMarker is implemented by types that can record warnings.
// Used by Exec to mark warnings without importing the tracker's package.
type WarningMarker interface {
	MarkWarning()
}

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

// GitDiffEnabledFromContext returns whether git diff is enabled in the context.
func GitDiffEnabledFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(gitDiffKey{}).(bool); ok {
		return v
	}
	return false
}

// CommitsCheckEnabledFromContext returns whether commits check is enabled in the context.
func CommitsCheckEnabledFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(commitsCheckKey{}).(bool); ok {
		return v
	}
	return false
}

// IsAutoExec returns whether auto execution mode is active.
func IsAutoExec(ctx context.Context) bool {
	v, _ := ctx.Value(autoExecKey{}).(bool)
	return v
}

// NameSuffixFromContext returns the name suffix from the context.
// Returns empty string if no suffix is set.
func NameSuffixFromContext(ctx context.Context) string {
	if s, ok := ctx.Value(nameSuffixKey{}).(string); ok {
		return s
	}
	return ""
}

// ForceRunFromContext returns whether forceRun is set in the context.
func ForceRunFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(forceRunKey{}).(bool); ok {
		return v
	}
	return false
}

// TaskFlagsFromContext retrieves resolved flag values from context.
func TaskFlagsFromContext(ctx context.Context) map[string]any {
	if flags, ok := ctx.Value(taskFlagsKey{}).(map[string]any); ok {
		return flags
	}
	return nil
}

// CLIFlagsFromContext retrieves CLI-provided flag values from context.
func CLIFlagsFromContext(ctx context.Context) map[string]any {
	if flags, ok := ctx.Value(cliFlagsKey{}).(map[string]any); ok {
		return flags
	}
	return nil
}

// NoticePatternsFromContext returns the notice patterns from context.
// Returns nil if not set (caller should use DefaultNoticePatterns).
func NoticePatternsFromContext(ctx context.Context) []string {
	if patterns, ok := ctx.Value(noticePatternsKey{}).([]string); ok {
		return patterns
	}
	return nil
}

// PlanFromContext returns the plan from the context as any.
// Returns nil if no plan is set.
func PlanFromContext(ctx context.Context) any {
	return ctx.Value(planKey{})
}

// TrackerFromContext returns the tracker from the context as any.
// Returns nil if no tracker is set.
func TrackerFromContext(ctx context.Context) any {
	return ctx.Value(trackerKey{})
}

// OutputFromContext returns the Output from the context.
// Returns nil if no output is set (caller should use StdOutput).
func OutputFromContext(ctx context.Context) *Output {
	if out, ok := ctx.Value(outputKey{}).(*Output); ok {
		return out
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Context Modifiers (Setters)
// ═══════════════════════════════════════════════════════════════════════════════

// ContextWithPath returns a new context with the given execution path.
func ContextWithPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, pathKey{}, path)
}

// ContextWithVerbose returns a new context with verbose mode set.
func ContextWithVerbose(ctx context.Context, verbose bool) context.Context {
	return context.WithValue(ctx, verboseKey{}, verbose)
}

// ContextWithGitDiffEnabled returns a new context with git diff enabled flag set.
func ContextWithGitDiffEnabled(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, gitDiffKey{}, enabled)
}

// ContextWithCommitsCheckEnabled returns a new context with commits check enabled flag set.
func ContextWithCommitsCheckEnabled(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, commitsCheckKey{}, enabled)
}

// ContextWithAutoExec returns a new context with auto execution mode enabled.
func ContextWithAutoExec(ctx context.Context) context.Context {
	return context.WithValue(ctx, autoExecKey{}, true)
}

// WithForceRun returns a new context with forceRun set to true.
func WithForceRun(ctx context.Context) context.Context {
	return context.WithValue(ctx, forceRunKey{}, true)
}

// ContextWithNameSuffix returns a new context with the given name suffix.
// Name suffixes are accumulated (e.g., nested calls combine).
func ContextWithNameSuffix(ctx context.Context, suffix string) context.Context {
	existing := NameSuffixFromContext(ctx)
	if existing != "" {
		suffix = existing + ":" + suffix
	}
	return context.WithValue(ctx, nameSuffixKey{}, suffix)
}

// WithTaskFlags stores resolved flag values in context.
func WithTaskFlags(ctx context.Context, flags map[string]any) context.Context {
	return context.WithValue(ctx, taskFlagsKey{}, flags)
}

// WithCLIFlags stores CLI-provided flag values in context.
func WithCLIFlags(ctx context.Context, flags map[string]any) context.Context {
	return context.WithValue(ctx, cliFlagsKey{}, flags)
}

// SetPlan stores the plan in the context.
func SetPlan(ctx context.Context, plan any) context.Context {
	return context.WithValue(ctx, planKey{}, plan)
}

// SetTracker stores the tracker in the context.
func SetTracker(ctx context.Context, tracker any) context.Context {
	return context.WithValue(ctx, trackerKey{}, tracker)
}

// SetOutput stores the output in the context.
func SetOutput(ctx context.Context, out *Output) context.Context {
	return context.WithValue(ctx, outputKey{}, out)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Environment Configuration
// ═══════════════════════════════════════════════════════════════════════════════

// EnvConfig holds environment variable overrides applied to Exec calls.
type EnvConfig struct {
	// Set contains environment variables to set (or replace). Keys are variable names.
	Set map[string]string
	// Filter contains prefixes of environment variables to remove.
	Filter []string
}

// ContextWithEnv returns a new context that sets an environment variable for Exec calls.
// The keyValue must be in "KEY=value" format.
func ContextWithEnv(ctx context.Context, keyValue string) context.Context {
	key, value, ok := strings.Cut(keyValue, "=")
	if !ok {
		return ctx
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
func ContextWithoutEnv(ctx context.Context, prefix string) context.Context {
	cfg := EnvConfigFromContext(ctx)
	cfg.Filter = append(cfg.Filter, prefix)
	return context.WithValue(ctx, envKey{}, cfg)
}

// EnvConfigFromContext returns a copy of the environment config from the context.
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
