package pk

import (
	"context"
	"maps"
)

// withFlagOverride returns a new context with a flag override for a specific task.
func withFlagOverride(ctx context.Context, taskName, flagName string, value any) context.Context {
	overrides := flagOverridesFromContext(ctx)
	newOverrides := make(map[string]map[string]any)

	// Shallow copy outer map.
	maps.Copy(newOverrides, overrides)

	// Shallow copy inner map for the specific task.
	inner := make(map[string]any)
	maps.Copy(inner, overrides[taskName])
	inner[flagName] = value
	newOverrides[taskName] = inner

	return context.WithValue(ctx, flagsKey, newOverrides)
}

// flagOverridesFromContext returns the map of task flag overrides from the context.
func flagOverridesFromContext(ctx context.Context) map[string]map[string]any {
	if v, ok := ctx.Value(flagsKey).(map[string]map[string]any); ok {
		return v
	}
	return nil
}
