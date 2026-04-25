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

func gitDiffEnabled(ctx context.Context) bool {
	v, _ := ctx.Value(ctxkey.GitDiff{}).(bool)
	return v
}

func commitsCheckEnabled(ctx context.Context) bool {
	v, _ := ctx.Value(ctxkey.CommitsCheck{}).(bool)
	return v
}

func cliFlagsFromContext(ctx context.Context) map[string]any {
	if flags, ok := ctx.Value(ctxkey.CLIFlags{}).(map[string]any); ok {
		return flags
	}
	return nil
}

func taskScopeFromEnv() string {
	scope := os.Getenv("TASK_SCOPE")
	if scope == "" || scope == "." {
		return ""
	}
	return scope
}
