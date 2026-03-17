package engine

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/fredrikaverpil/pocket/pk/internal/ctxkey"
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

// GitDiffEnabledFromContext returns whether git diff is enabled in the context.
func GitDiffEnabledFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(ctxkey.GitDiff{}).(bool); ok {
		return v
	}
	return false
}

// CommitsCheckEnabledFromContext returns whether commits check is enabled in the context.
func CommitsCheckEnabledFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(ctxkey.CommitsCheck{}).(bool); ok {
		return v
	}
	return false
}

// IsAutoExec returns whether auto execution mode is active.
func IsAutoExec(ctx context.Context) bool {
	v, _ := ctx.Value(ctxkey.AutoExec{}).(bool)
	return v
}

// NameSuffixFromContext returns the name suffix from the context.
// Returns empty string if no suffix is set.
func NameSuffixFromContext(ctx context.Context) string {
	if s, ok := ctx.Value(ctxkey.NameSuffix{}).(string); ok {
		return s
	}
	return ""
}

// ForceRunFromContext returns whether forceRun is set in the context.
func ForceRunFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(ctxkey.ForceRun{}).(bool); ok {
		return v
	}
	return false
}

// TaskFlagsFromContext retrieves resolved flag values from context.
func TaskFlagsFromContext(ctx context.Context) map[string]any {
	if flags, ok := ctx.Value(ctxkey.TaskFlags{}).(map[string]any); ok {
		return flags
	}
	return nil
}

// CLIFlagsFromContext retrieves CLI-provided flag values from context.
func CLIFlagsFromContext(ctx context.Context) map[string]any {
	if flags, ok := ctx.Value(ctxkey.CLIFlags{}).(map[string]any); ok {
		return flags
	}
	return nil
}

// NoticePatternsFromContext returns the notice patterns from context.
// Returns nil if not set (caller should use DefaultNoticePatterns).
func NoticePatternsFromContext(ctx context.Context) []string {
	if patterns, ok := ctx.Value(ctxkey.NoticePatterns{}).([]string); ok {
		return patterns
	}
	return nil
}

// PlanFromContext returns the plan from the context as any.
// Returns nil if no plan is set.
func PlanFromContext(ctx context.Context) any {
	return ctx.Value(ctxkey.Plan{})
}

// TrackerFromContext returns the tracker from the context as any.
// Returns nil if no tracker is set.
func TrackerFromContext(ctx context.Context) any {
	return ctx.Value(ctxkey.Tracker{})
}

// OutputFromContext returns the Output from the context.
// Returns nil if no output is set (caller should use StdOutput).
func OutputFromContext(ctx context.Context) *Output {
	if out, ok := ctx.Value(ctxkey.Output{}).(*Output); ok {
		return out
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Context Modifiers (Setters)
// ═══════════════════════════════════════════════════════════════════════════════

// ContextWithPath returns a new context with the given execution path.
func ContextWithPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, ctxkey.Path{}, path)
}

// ContextWithVerbose returns a new context with verbose mode set.
func ContextWithVerbose(ctx context.Context, verbose bool) context.Context {
	return context.WithValue(ctx, ctxkey.Verbose{}, verbose)
}

// ContextWithGitDiffEnabled returns a new context with git diff enabled flag set.
func ContextWithGitDiffEnabled(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, ctxkey.GitDiff{}, enabled)
}

// ContextWithCommitsCheckEnabled returns a new context with commits check enabled flag set.
func ContextWithCommitsCheckEnabled(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, ctxkey.CommitsCheck{}, enabled)
}

// ContextWithAutoExec returns a new context with auto execution mode enabled.
func ContextWithAutoExec(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxkey.AutoExec{}, true)
}

// WithForceRun returns a new context with forceRun set to true.
func WithForceRun(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxkey.ForceRun{}, true)
}

// ContextWithNameSuffix returns a new context with the given name suffix.
// Name suffixes are accumulated (e.g., nested calls combine).
func ContextWithNameSuffix(ctx context.Context, suffix string) context.Context {
	existing := NameSuffixFromContext(ctx)
	if existing != "" {
		suffix = existing + ":" + suffix
	}
	return context.WithValue(ctx, ctxkey.NameSuffix{}, suffix)
}

// WithTaskFlags stores resolved flag values in context.
func WithTaskFlags(ctx context.Context, flags map[string]any) context.Context {
	return context.WithValue(ctx, ctxkey.TaskFlags{}, flags)
}

// WithCLIFlags stores CLI-provided flag values in context.
func WithCLIFlags(ctx context.Context, flags map[string]any) context.Context {
	return context.WithValue(ctx, ctxkey.CLIFlags{}, flags)
}

// SetPlan stores the plan in the context.
func SetPlan(ctx context.Context, plan any) context.Context {
	return context.WithValue(ctx, ctxkey.Plan{}, plan)
}

// SetTracker stores the tracker in the context.
func SetTracker(ctx context.Context, tracker any) context.Context {
	return context.WithValue(ctx, ctxkey.Tracker{}, tracker)
}

// SetOutput stores the output in the context.
func SetOutput(ctx context.Context, out *Output) context.Context {
	return context.WithValue(ctx, ctxkey.Output{}, out)
}

// SetNoticePatterns stores custom notice patterns in the context.
func SetNoticePatterns(ctx context.Context, patterns []string) context.Context {
	return context.WithValue(ctx, ctxkey.NoticePatterns{}, patterns)
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
	return context.WithValue(ctx, ctxkey.Env{}, cfg)
}

// ContextWithoutEnv returns a new context that filters out environment variables
// matching the given prefix from Exec calls.
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
