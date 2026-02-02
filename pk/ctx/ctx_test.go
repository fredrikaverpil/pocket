package ctx

import (
	"context"
	"reflect"
	"testing"
)

func TestPathContext(t *testing.T) {
	ctx := context.Background()

	// Default should return "."
	if path := PathFromContext(ctx); path != "." {
		t.Errorf("expected default path to be \".\", got %q", path)
	}

	// Set path and retrieve it
	ctx = WithPath(ctx, "services/api")
	if path := PathFromContext(ctx); path != "services/api" {
		t.Errorf("expected path to be \"services/api\", got %q", path)
	}
}

func TestVerboseContext(t *testing.T) {
	ctx := context.Background()

	// Default should be false
	if Verbose(ctx) {
		t.Error("expected verbose to be false by default")
	}

	// Set verbose and check
	ctx = WithVerbose(ctx, true)
	if !Verbose(ctx) {
		t.Error("expected verbose to be true after WithVerbose(true)")
	}

	// Set to false explicitly
	ctx = WithVerbose(ctx, false)
	if Verbose(ctx) {
		t.Error("expected verbose to be false after WithVerbose(false)")
	}
}

func TestGitDiffEnabledContext(t *testing.T) {
	ctx := context.Background()

	// Default should be false
	if GitDiffEnabledFromContext(ctx) {
		t.Error("expected git diff to be disabled by default")
	}

	// Enable git diff
	ctx = WithGitDiffEnabled(ctx, true)
	if !GitDiffEnabledFromContext(ctx) {
		t.Error("expected git diff to be enabled after WithGitDiffEnabled(true)")
	}
}

func TestAutoExecContext(t *testing.T) {
	ctx := context.Background()

	// Default should be false
	if IsAutoExec(ctx) {
		t.Error("expected auto exec to be false by default")
	}

	// Enable auto exec
	ctx = WithAutoExec(ctx)
	if !IsAutoExec(ctx) {
		t.Error("expected auto exec to be true after WithAutoExec")
	}
}

func TestEnvContext(t *testing.T) {
	ctx := context.Background()

	// Default should be empty config
	cfg := EnvConfigFromContext(ctx)
	if cfg.Set != nil || len(cfg.Filter) != 0 {
		t.Error("expected empty config by default")
	}

	// Set environment variable
	ctx = WithEnv(ctx, "FOO=bar")
	cfg = EnvConfigFromContext(ctx)
	if cfg.Set["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got %v", cfg.Set)
	}

	// Set another variable
	ctx = WithEnv(ctx, "BAZ=qux")
	cfg = EnvConfigFromContext(ctx)
	if cfg.Set["FOO"] != "bar" || cfg.Set["BAZ"] != "qux" {
		t.Errorf("expected both variables, got %v", cfg.Set)
	}

	// Replace existing variable
	ctx = WithEnv(ctx, "FOO=replaced")
	cfg = EnvConfigFromContext(ctx)
	if cfg.Set["FOO"] != "replaced" {
		t.Errorf("expected FOO to be replaced, got %v", cfg.Set)
	}

	// Invalid format should be ignored
	ctx = WithEnv(ctx, "INVALID")
	cfg = EnvConfigFromContext(ctx)
	if _, exists := cfg.Set["INVALID"]; exists {
		t.Error("expected invalid format to be ignored")
	}

	// Test filtering
	ctx = WithoutEnv(ctx, "VIRTUAL_ENV")
	cfg = EnvConfigFromContext(ctx)
	if len(cfg.Filter) != 1 || cfg.Filter[0] != "VIRTUAL_ENV" {
		t.Errorf("expected VIRTUAL_ENV in filter, got %v", cfg.Filter)
	}

	// Add another filter
	ctx = WithoutEnv(ctx, "CONDA")
	cfg = EnvConfigFromContext(ctx)
	if len(cfg.Filter) != 2 {
		t.Errorf("expected 2 filters, got %d", len(cfg.Filter))
	}
}

func TestEnvConfigImmutability(t *testing.T) {
	ctx := context.Background()
	ctx = WithEnv(ctx, "FOO=bar")

	// Get config twice
	cfg1 := EnvConfigFromContext(ctx)
	cfg2 := EnvConfigFromContext(ctx)

	// Modify one
	cfg1.Set["MODIFIED"] = "value"

	// The other should be unaffected
	if _, exists := cfg2.Set["MODIFIED"]; exists {
		t.Error("expected EnvConfigFromContext to return a copy, not the original")
	}
}

func TestNameSuffixContext(t *testing.T) {
	ctx := context.Background()

	// Default should be empty string
	if suffix := NameSuffixFromContext(ctx); suffix != "" {
		t.Errorf("expected empty suffix by default, got %q", suffix)
	}

	// Set suffix
	ctx = WithNameSuffix(ctx, "3.9")
	if suffix := NameSuffixFromContext(ctx); suffix != "3.9" {
		t.Errorf("expected suffix to be \"3.9\", got %q", suffix)
	}

	// Nested suffix should accumulate
	ctx = WithNameSuffix(ctx, "linux")
	if suffix := NameSuffixFromContext(ctx); suffix != "3.9:linux" {
		t.Errorf("expected accumulated suffix \"3.9:linux\", got %q", suffix)
	}
}

func TestForceRunContext(t *testing.T) {
	ctx := context.Background()

	// Default should be false
	if ForceRunFromContext(ctx) {
		t.Error("expected forceRun to be false by default")
	}

	// Set forceRun and check
	ctx = WithForceRun(ctx)
	if !ForceRunFromContext(ctx) {
		t.Error("expected forceRun to be true after WithForceRun")
	}
}

func TestEnvConfigClone(t *testing.T) {
	ctx := context.Background()
	ctx = WithEnv(ctx, "A=1")
	ctx = WithoutEnv(ctx, "PREFIX")

	cfg := EnvConfigFromContext(ctx)

	// Verify we got what we set
	if cfg.Set["A"] != "1" {
		t.Error("expected A=1")
	}
	if !reflect.DeepEqual(cfg.Filter, []string{"PREFIX"}) {
		t.Errorf("expected Filter=[PREFIX], got %v", cfg.Filter)
	}

	// Mutate the returned config
	cfg.Set["B"] = "2"
	cfg.Filter = append(cfg.Filter, "EXTRA")

	// Get again and verify mutations didn't affect context
	cfg2 := EnvConfigFromContext(ctx)
	if _, exists := cfg2.Set["B"]; exists {
		t.Error("mutation of Set should not affect context")
	}
	if len(cfg2.Filter) != 1 {
		t.Error("mutation of Filter should not affect context")
	}
}
