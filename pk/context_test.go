package pk

import (
	"context"
	"testing"
)

func TestContextWithNameSuffix(t *testing.T) {
	t.Run("Single", func(t *testing.T) {
		ctx := contextWithNameSuffix(context.Background(), "3.9")
		if got := nameSuffixFromContext(ctx); got != "3.9" {
			t.Errorf("expected %q, got %q", "3.9", got)
		}
	})

	t.Run("Accumulation", func(t *testing.T) {
		ctx := contextWithNameSuffix(context.Background(), "a")
		ctx = contextWithNameSuffix(ctx, "b")
		if got := nameSuffixFromContext(ctx); got != "a:b" {
			t.Errorf("expected %q, got %q", "a:b", got)
		}
	})

	t.Run("TripleNesting", func(t *testing.T) {
		ctx := contextWithNameSuffix(context.Background(), "x")
		ctx = contextWithNameSuffix(ctx, "y")
		ctx = contextWithNameSuffix(ctx, "z")
		if got := nameSuffixFromContext(ctx); got != "x:y:z" {
			t.Errorf("expected %q, got %q", "x:y:z", got)
		}
	})
}

func TestContextWithEnv(t *testing.T) {
	t.Run("ValidKeyValue", func(t *testing.T) {
		ctx := ContextWithEnv(context.Background(), "MY_VAR=hello")
		cfg := EnvConfigFromContext(ctx)
		if cfg.Set["MY_VAR"] != "hello" {
			t.Errorf("expected MY_VAR=hello, got %v", cfg.Set)
		}
	})

	t.Run("Accumulation", func(t *testing.T) {
		ctx := ContextWithEnv(context.Background(), "A=1")
		ctx = ContextWithEnv(ctx, "B=2")
		cfg := EnvConfigFromContext(ctx)
		if cfg.Set["A"] != "1" || cfg.Set["B"] != "2" {
			t.Errorf("expected A=1, B=2, got %v", cfg.Set)
		}
	})

	t.Run("OverwriteSameKey", func(t *testing.T) {
		ctx := ContextWithEnv(context.Background(), "X=old")
		ctx = ContextWithEnv(ctx, "X=new")
		cfg := EnvConfigFromContext(ctx)
		if cfg.Set["X"] != "new" {
			t.Errorf("expected X=new, got %v", cfg.Set)
		}
	})

	t.Run("InvalidFormat", func(t *testing.T) {
		ctx := ContextWithEnv(context.Background(), "NOEQUALSSIGN")
		cfg := EnvConfigFromContext(ctx)
		if len(cfg.Set) != 0 {
			t.Errorf("expected empty set for invalid format, got %v", cfg.Set)
		}
	})
}

func TestContextWithoutEnv(t *testing.T) {
	t.Run("SingleFilter", func(t *testing.T) {
		ctx := ContextWithoutEnv(context.Background(), "VIRTUAL_ENV")
		cfg := EnvConfigFromContext(ctx)
		if len(cfg.Filter) != 1 || cfg.Filter[0] != "VIRTUAL_ENV" {
			t.Errorf("expected [VIRTUAL_ENV], got %v", cfg.Filter)
		}
	})

	t.Run("Accumulation", func(t *testing.T) {
		ctx := ContextWithoutEnv(context.Background(), "A")
		ctx = ContextWithoutEnv(ctx, "B")
		cfg := EnvConfigFromContext(ctx)
		if len(cfg.Filter) != 2 {
			t.Errorf("expected 2 filters, got %v", cfg.Filter)
		}
	})
}

func TestEnvConfigFromContext(t *testing.T) {
	t.Run("DefaultEmpty", func(t *testing.T) {
		cfg := EnvConfigFromContext(context.Background())
		if cfg.Set != nil {
			t.Errorf("expected nil Set, got %v", cfg.Set)
		}
		if cfg.Filter != nil {
			t.Errorf("expected nil Filter, got %v", cfg.Filter)
		}
	})

	t.Run("DefensiveCopy", func(t *testing.T) {
		ctx := ContextWithEnv(context.Background(), "A=1")
		cfg1 := EnvConfigFromContext(ctx)
		cfg2 := EnvConfigFromContext(ctx)

		// Mutating cfg1 should not affect cfg2.
		cfg1.Set["A"] = "mutated"
		if cfg2.Set["A"] != "1" {
			t.Error("EnvConfigFromContext should return defensive copies")
		}
	})
}

func TestVerbose(t *testing.T) {
	t.Run("DefaultFalse", func(t *testing.T) {
		if Verbose(context.Background()) {
			t.Error("expected false by default")
		}
	})

	t.Run("SetTrue", func(t *testing.T) {
		ctx := contextWithVerbose(context.Background(), true)
		if !Verbose(ctx) {
			t.Error("expected true after setting")
		}
	})
}

func TestIsAutoExec(t *testing.T) {
	t.Run("DefaultFalse", func(t *testing.T) {
		if isAutoExec(context.Background()) {
			t.Error("expected false by default")
		}
	})

	t.Run("SetTrue", func(t *testing.T) {
		ctx := contextWithAutoExec(context.Background())
		if !isAutoExec(ctx) {
			t.Error("expected true after setting")
		}
	})
}

func TestPathFromContext(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		if got := PathFromContext(context.Background()); got != "." {
			t.Errorf("expected %q, got %q", ".", got)
		}
	})

	t.Run("Set", func(t *testing.T) {
		ctx := ContextWithPath(context.Background(), "services/api")
		if got := PathFromContext(ctx); got != "services/api" {
			t.Errorf("expected %q, got %q", "services/api", got)
		}
	})
}
