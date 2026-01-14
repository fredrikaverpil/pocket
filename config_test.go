package pocket

import (
	"context"
	"testing"
)

func TestConfig_WithDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       Config
		wantShimName string
		wantPosix    bool
	}{
		{
			name:         "empty config gets default shim name and posix",
			config:       Config{},
			wantShimName: "pok",
			wantPosix:    true,
		},
		{
			name: "custom shim name is preserved",
			config: Config{
				Shim: &ShimConfig{Name: "build", Posix: true},
			},
			wantShimName: "build",
			wantPosix:    true,
		},
		{
			name: "empty shim name gets default",
			config: Config{
				Shim: &ShimConfig{Posix: true},
			},
			wantShimName: "pok",
			wantPosix:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.config.WithDefaults()

			if got.Shim == nil {
				t.Fatal("WithDefaults().Shim is nil")
			}
			if got.Shim.Name != tt.wantShimName {
				t.Errorf("WithDefaults().Shim.Name = %q, want %q", got.Shim.Name, tt.wantShimName)
			}
		})
	}
}

func TestSerial_Funcs(t *testing.T) {
	t.Parallel()

	fn1 := Func("test-format", "format test files", func(_ context.Context) error { return nil })
	fn2 := Func("test-lint", "lint test files", func(_ context.Context) error { return nil })

	runnable := Serial(fn1, fn2)
	// Check funcs returns both funcs.
	funcs := runnable.funcs()
	if len(funcs) != 2 {
		t.Errorf("funcs() length = %d, want 2", len(funcs))
	}
}

func TestParallel_Funcs(t *testing.T) {
	t.Parallel()

	fn1 := Func("fn1", "func 1", func(_ context.Context) error { return nil })
	fn2 := Func("fn2", "func 2", func(_ context.Context) error { return nil })

	runnable := Parallel(fn1, fn2)
	funcs := runnable.funcs()
	if len(funcs) != 2 {
		t.Errorf("funcs() length = %d, want 2", len(funcs))
	}
}

func TestConfig_AutoRun(t *testing.T) {
	t.Parallel()

	fn1 := Func("deploy", "deploy app", func(_ context.Context) error { return nil })
	fn2 := Func("release", "release app", func(_ context.Context) error { return nil })

	cfg := Config{
		AutoRun: Serial(fn1, fn2),
	}

	funcs := cfg.AutoRun.funcs()
	if len(funcs) != 2 {
		t.Errorf("AutoRun.funcs() length = %d, want 2", len(funcs))
	}
}

func TestNested_Serial_Parallel(t *testing.T) {
	t.Parallel()

	fn1 := Func("fn1", "func 1", func(_ context.Context) error { return nil })
	fn2 := Func("fn2", "func 2", func(_ context.Context) error { return nil })
	fn3 := Func("fn3", "func 3", func(_ context.Context) error { return nil })

	runnable := Serial(
		fn1,
		Parallel(fn2, fn3),
	)
	funcs := runnable.funcs()
	if len(funcs) != 3 {
		t.Errorf("funcs() length = %d, want 3", len(funcs))
	}
}
