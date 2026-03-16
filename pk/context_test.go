package pk

import (
	"context"
	"testing"

	"github.com/fredrikaverpil/pocket/pk/internal/engine"
)

func TestContextWithNameSuffix(t *testing.T) {
	t.Run("Single", func(t *testing.T) {
		ctx := engine.ContextWithNameSuffix(context.Background(), "3.9")
		if got := engine.NameSuffixFromContext(ctx); got != "3.9" {
			t.Errorf("expected %q, got %q", "3.9", got)
		}
	})

	t.Run("Accumulation", func(t *testing.T) {
		ctx := engine.ContextWithNameSuffix(context.Background(), "a")
		ctx = engine.ContextWithNameSuffix(ctx, "b")
		if got := engine.NameSuffixFromContext(ctx); got != "a:b" {
			t.Errorf("expected %q, got %q", "a:b", got)
		}
	})

	t.Run("TripleNesting", func(t *testing.T) {
		ctx := engine.ContextWithNameSuffix(context.Background(), "x")
		ctx = engine.ContextWithNameSuffix(ctx, "y")
		ctx = engine.ContextWithNameSuffix(ctx, "z")
		if got := engine.NameSuffixFromContext(ctx); got != "x:y:z" {
			t.Errorf("expected %q, got %q", "x:y:z", got)
		}
	})
}

func TestContextWithEnv(t *testing.T) {
	t.Run("ValidKeyValue", func(t *testing.T) {
		ctx := engine.ContextWithEnv(context.Background(), "MY_VAR=hello")
		cfg := engine.EnvConfigFromContext(ctx)
		if cfg.Set["MY_VAR"] != "hello" {
			t.Errorf("expected MY_VAR=hello, got %v", cfg.Set)
		}
	})

	t.Run("Accumulation", func(t *testing.T) {
		ctx := engine.ContextWithEnv(context.Background(), "A=1")
		ctx = engine.ContextWithEnv(ctx, "B=2")
		cfg := engine.EnvConfigFromContext(ctx)
		if cfg.Set["A"] != "1" || cfg.Set["B"] != "2" {
			t.Errorf("expected A=1, B=2, got %v", cfg.Set)
		}
	})

	t.Run("OverwriteSameKey", func(t *testing.T) {
		ctx := engine.ContextWithEnv(context.Background(), "X=old")
		ctx = engine.ContextWithEnv(ctx, "X=new")
		cfg := engine.EnvConfigFromContext(ctx)
		if cfg.Set["X"] != "new" {
			t.Errorf("expected X=new, got %v", cfg.Set)
		}
	})

	t.Run("InvalidFormat", func(t *testing.T) {
		ctx := engine.ContextWithEnv(context.Background(), "NOEQUALSSIGN")
		cfg := engine.EnvConfigFromContext(ctx)
		if len(cfg.Set) != 0 {
			t.Errorf("expected empty set for invalid format, got %v", cfg.Set)
		}
	})
}

func TestContextWithoutEnv(t *testing.T) {
	t.Run("SingleFilter", func(t *testing.T) {
		ctx := engine.ContextWithoutEnv(context.Background(), "VIRTUAL_ENV")
		cfg := engine.EnvConfigFromContext(ctx)
		if len(cfg.Filter) != 1 || cfg.Filter[0] != "VIRTUAL_ENV" {
			t.Errorf("expected [VIRTUAL_ENV], got %v", cfg.Filter)
		}
	})

	t.Run("Accumulation", func(t *testing.T) {
		ctx := engine.ContextWithoutEnv(context.Background(), "A")
		ctx = engine.ContextWithoutEnv(ctx, "B")
		cfg := engine.EnvConfigFromContext(ctx)
		if len(cfg.Filter) != 2 {
			t.Errorf("expected 2 filters, got %v", cfg.Filter)
		}
	})
}

func TestEnvConfigFromContext(t *testing.T) {
	t.Run("DefaultEmpty", func(t *testing.T) {
		cfg := engine.EnvConfigFromContext(context.Background())
		if cfg.Set != nil {
			t.Errorf("expected nil Set, got %v", cfg.Set)
		}
		if cfg.Filter != nil {
			t.Errorf("expected nil Filter, got %v", cfg.Filter)
		}
	})

	t.Run("DefensiveCopy", func(t *testing.T) {
		ctx := engine.ContextWithEnv(context.Background(), "A=1")
		cfg1 := engine.EnvConfigFromContext(ctx)
		cfg2 := engine.EnvConfigFromContext(ctx)

		// Mutating cfg1 should not affect cfg2.
		cfg1.Set["A"] = "mutated"
		if cfg2.Set["A"] != "1" {
			t.Error("EnvConfigFromContext should return defensive copies")
		}
	})
}

func TestVerbose(t *testing.T) {
	t.Run("DefaultFalse", func(t *testing.T) {
		if engine.Verbose(context.Background()) {
			t.Error("expected false by default")
		}
	})

	t.Run("SetTrue", func(t *testing.T) {
		ctx := engine.ContextWithVerbose(context.Background(), true)
		if !engine.Verbose(ctx) {
			t.Error("expected true after setting")
		}
	})
}

func TestIsAutoExec(t *testing.T) {
	t.Run("DefaultFalse", func(t *testing.T) {
		if engine.IsAutoExec(context.Background()) {
			t.Error("expected false by default")
		}
	})

	t.Run("SetTrue", func(t *testing.T) {
		ctx := engine.ContextWithAutoExec(context.Background())
		if !engine.IsAutoExec(ctx) {
			t.Error("expected true after setting")
		}
	})
}

func TestPathFromContext(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		if got := engine.PathFromContext(context.Background()); got != "." {
			t.Errorf("expected %q, got %q", ".", got)
		}
	})

	t.Run("Set", func(t *testing.T) {
		ctx := engine.ContextWithPath(context.Background(), "services/api")
		if got := engine.PathFromContext(ctx); got != "services/api" {
			t.Errorf("expected %q, got %q", "services/api", got)
		}
	})
}
