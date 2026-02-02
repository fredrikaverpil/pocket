package pk

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk/ctx"
)

// Re-export context keys for internal use by pk package.
const (
	planKey     = ctx.PlanKey
	trackerKey  = ctx.TrackerKey
	forceRunKey = ctx.ForceRunKey
	outputKey   = ctx.OutputKey
	gitDiffKey  = ctx.GitDiffKey
	autoExecKey = ctx.AutoExecKey
)

// Re-export EnvConfig type alias for internal use.
type envConfig = ctx.EnvConfig

// WithPath returns a new context with the given path set.
// The path is relative to the git root.
func WithPath(c context.Context, path string) context.Context {
	return ctx.WithPath(c, path)
}

// PathFromContext returns the current execution path from the context.
// Returns "." if no path is set (meaning git root).
func PathFromContext(c context.Context) string {
	return ctx.PathFromContext(c)
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
func WithVerbose(c context.Context, verbose bool) context.Context {
	return ctx.WithVerbose(c, verbose)
}

// Verbose returns whether verbose mode is enabled in the context.
func Verbose(c context.Context) bool {
	return ctx.Verbose(c)
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

// WithEnv returns a new context that sets an environment variable.
// The keyValue must be in "KEY=value" format.
// If a variable with the same key already exists, it is replaced.
func WithEnv(c context.Context, keyValue string) context.Context {
	return ctx.WithEnv(c, keyValue)
}

// WithoutEnv returns a new context that filters out environment variables
// matching the given prefix. For example, WithoutEnv(ctx, "VIRTUAL_ENV")
// removes VIRTUAL_ENV from the environment.
func WithoutEnv(c context.Context, prefix string) context.Context {
	return ctx.WithoutEnv(c, prefix)
}

// envConfigFromContext returns the environment config from the context.
// Returns a copy to avoid mutating the original.
func envConfigFromContext(c context.Context) envConfig {
	return ctx.EnvConfigFromContext(c)
}

// withNameSuffix returns a new context with the given name suffix.
// Name suffixes are accumulated (e.g., nested WithNameSuffix calls combine).
func withNameSuffix(c context.Context, suffix string) context.Context {
	return ctx.WithNameSuffix(c, suffix)
}

// nameSuffixFromContext returns the name suffix from the context.
// Returns empty string if no suffix is set.
func nameSuffixFromContext(c context.Context) string {
	return ctx.NameSuffixFromContext(c)
}
