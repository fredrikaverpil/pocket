package pk

import (
	"context"
	"os"

	"github.com/fredrikaverpil/pocket/pk/internal/ctxkey"
)

func nameSuffixFromContext(ctx context.Context) string {
	if s, ok := ctx.Value(ctxkey.NameSuffix{}).(string); ok {
		return s
	}
	return ""
}

func contextWithNameSuffix(ctx context.Context, suffix string) context.Context {
	existing := nameSuffixFromContext(ctx)
	if existing != "" {
		suffix = existing + ":" + suffix
	}
	return context.WithValue(ctx, ctxkey.NameSuffix{}, suffix)
}

func isAutoExec(ctx context.Context) bool {
	v, _ := ctx.Value(ctxkey.AutoExec{}).(bool)
	return v
}

func forceRunFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(ctxkey.ForceRun{}).(bool)
	return v
}

func serialFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(ctxkey.Serial{}).(bool)
	return v
}

func gitDiffEnabled(ctx context.Context) bool {
	v, _ := ctx.Value(ctxkey.GitDiff{}).(bool)
	return v
}

func commitsCheckEnabled(ctx context.Context) bool {
	v, _ := ctx.Value(ctxkey.CommitsCheck{}).(bool)
	return v
}

// cliFlagOverrides carries CLI-provided flag overrides together with the name
// of the task they were aimed at. Scoping by target prevents the overrides from
// leaking into subtasks composed within the invoked task's body that happen to
// declare same-named flags.
type cliFlagOverrides struct {
	target string
	flags  map[string]any
}

func withCLIFlags(ctx context.Context, target string, flags map[string]any) context.Context {
	return context.WithValue(ctx, ctxkey.CLIFlags{}, cliFlagOverrides{target: target, flags: flags})
}

// cliFlagsForTask returns the CLI flag overrides only if they were aimed at
// effectiveName, otherwise nil.
func cliFlagsForTask(ctx context.Context, effectiveName string) map[string]any {
	o, ok := ctx.Value(ctxkey.CLIFlags{}).(cliFlagOverrides)
	if !ok || o.target != effectiveName {
		return nil
	}
	return o.flags
}

func taskScopeFromEnv() string {
	scope := os.Getenv("TASK_SCOPE")
	if scope == "" || scope == "." {
		return ""
	}
	return scope
}
