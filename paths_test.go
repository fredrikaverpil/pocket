package pocket

import (
	"context"
	"testing"
)

func TestPaths_In(t *testing.T) {
	fn := Func("test", "test", func(_ context.Context) error { return nil })
	p := Paths(fn).In("proj1", "proj2")

	resolved := p.Resolve()
	if len(resolved) != 2 {
		t.Errorf("expected 2 paths, got %d: %v", len(resolved), resolved)
	}
	if !p.RunsIn("proj1") {
		t.Error("expected RunsIn(proj1) to be true")
	}
	if !p.RunsIn("proj2") {
		t.Error("expected RunsIn(proj2) to be true")
	}
	if p.RunsIn("proj3") {
		t.Error("expected RunsIn(proj3) to be false")
	}
}

func TestPaths_Except(t *testing.T) {
	fn := Func("test", "test", func(_ context.Context) error { return nil })
	p := Paths(fn).DetectBy(func() []string {
		return []string{"proj1", "proj2", "vendor"}
	}).Except("vendor")

	resolved := p.Resolve()
	if len(resolved) != 2 {
		t.Errorf("expected 2 paths, got %d: %v", len(resolved), resolved)
	}
	if p.RunsIn("vendor") {
		t.Error("expected vendor to be excluded")
	}
}

func TestPaths_DetectBy(t *testing.T) {
	fn := Func("test", "test", func(_ context.Context) error { return nil })
	p := Paths(fn).DetectBy(func() []string {
		return []string{"a", "b", "c"}
	})

	resolved := p.Resolve()
	if len(resolved) != 3 {
		t.Errorf("expected 3 paths, got %d: %v", len(resolved), resolved)
	}
}

func TestPaths_DetectBy_ReturnsCorrectPaths(t *testing.T) {
	fn := Func("test", "test", func(_ context.Context) error { return nil })
	p := Paths(fn).DetectBy(func() []string {
		return []string{"mod1", "mod2"}
	})

	resolved := p.Resolve()
	if len(resolved) != 2 {
		t.Errorf("expected 2 paths, got %d: %v", len(resolved), resolved)
	}
}

func TestPaths_NoDetect_ReturnsEmpty(t *testing.T) {
	fn := Func("test", "test", func(_ context.Context) error { return nil })
	p := Paths(fn) // no detection set

	resolved := p.Resolve()
	if len(resolved) != 0 {
		t.Errorf("expected 0 paths, got %d: %v", len(resolved), resolved)
	}
}

func TestPaths_CombineDetectAndInclude(t *testing.T) {
	fn := Func("test", "test", func(_ context.Context) error { return nil })
	p := Paths(fn).DetectBy(func() []string {
		return []string{"detected1", "detected2"}
	}).In("detected1") // filter to only detected1

	resolved := p.Resolve()
	if len(resolved) != 1 {
		t.Errorf("expected 1 path, got %d: %v", len(resolved), resolved)
	}
	if !p.RunsIn("detected1") {
		t.Error("expected RunsIn(detected1) to be true")
	}
}

func TestPaths_Immutability(t *testing.T) {
	fn := Func("test", "test", func(_ context.Context) error { return nil })
	p1 := Paths(fn).In("proj1")
	p2 := p1.In("proj2")

	// p1 should still only have proj1
	if p1.RunsIn("proj2") {
		t.Error("p1 should not include proj2 (immutability violated)")
	}
	// p2 should have both
	if !p2.RunsIn("proj1") || !p2.RunsIn("proj2") {
		t.Error("p2 should include both proj1 and proj2")
	}
}

func TestPaths_Funcs(t *testing.T) {
	fn := Func("test-func", "test func", func(_ context.Context) error { return nil })
	p := Paths(fn).In(".")

	funcs := p.funcs()
	if len(funcs) != 1 {
		t.Errorf("expected 1 func, got %d", len(funcs))
	}
	if funcs[0].name != "test-func" {
		t.Errorf("expected func name 'test-func', got %s", funcs[0].name)
	}
}

func TestPaths_RegexPatterns(t *testing.T) {
	fn := Func("test", "test", func(_ context.Context) error { return nil })
	p := Paths(fn).DetectBy(func() []string {
		return []string{"services/api", "services/web", "tools/cli"}
	}).In("services/.*")

	resolved := p.Resolve()
	if len(resolved) != 2 {
		t.Errorf("expected 2 paths, got %d: %v", len(resolved), resolved)
	}
	if !p.RunsIn("services/api") {
		t.Error("expected services/api to match")
	}
	if p.RunsIn("tools/cli") {
		t.Error("expected tools/cli to not match")
	}
}

func TestPaths_RootOnly(t *testing.T) {
	fn := Func("test", "test", func(_ context.Context) error { return nil })
	p := Paths(fn).In(".")

	if !p.RunsIn(".") {
		t.Error("expected RunsIn(.) to be true")
	}
	if p.RunsIn("subdir") {
		t.Error("expected RunsIn(subdir) to be false")
	}
}

func TestCollectPathMappings(t *testing.T) {
	fn1 := Func("fn1", "func 1", func(_ context.Context) error { return nil })
	fn2 := Func("fn2", "func 2", func(_ context.Context) error { return nil })
	fn3 := Func("fn3", "func 3", func(_ context.Context) error { return nil })

	// Wrap fn1 and fn2 with Paths().DetectBy().
	wrapped := Paths(Serial(fn1, fn2)).DetectBy(func() []string {
		return []string{"proj1", "proj2"}
	})

	// Create a serial runnable with wrapped and unwrapped funcs.
	runnable := Serial(wrapped, fn3)
	mappings := CollectPathMappings(runnable)

	// fn1 and fn2 should be in mappings (from wrapped Paths).
	if _, ok := mappings["fn1"]; !ok {
		t.Error("expected fn1 to be in mappings")
	}
	if _, ok := mappings["fn2"]; !ok {
		t.Error("expected fn2 to be in mappings")
	}
	// fn3 should NOT be in mappings (not wrapped with Paths).
	if _, ok := mappings["fn3"]; ok {
		t.Error("expected fn3 to NOT be in mappings")
	}

	// Check that the paths resolve correctly.
	if paths, ok := mappings["fn1"]; ok {
		resolved := paths.Resolve()
		if len(resolved) != 2 {
			t.Errorf("expected 2 resolved paths, got %d", len(resolved))
		}
	}
}

func TestCollectModuleDirectories(t *testing.T) {
	fn := Func("test", "test", func(_ context.Context) error { return nil })
	wrapped := Paths(fn).DetectBy(func() []string {
		return []string{"proj1", "proj2"}
	})

	dirs := CollectModuleDirectories(wrapped)

	// Should include root (.) and detected directories.
	if len(dirs) != 3 {
		t.Errorf("expected 3 directories, got %d: %v", len(dirs), dirs)
	}

	// Check specific directories.
	expected := map[string]bool{".": true, "proj1": true, "proj2": true}
	for _, d := range dirs {
		if !expected[d] {
			t.Errorf("unexpected directory: %s", d)
		}
	}
}

func TestPaths_Skip(t *testing.T) {
	fn1 := Func("test-format", "format code", func(_ context.Context) error { return nil })
	fn2 := Func("test-lint", "lint code", func(_ context.Context) error { return nil })
	fn3 := Func("test-build", "build code", func(_ context.Context) error { return nil })

	p := Paths(Serial(fn1, fn2, fn3)).In(".").Skip(fn2)

	// funcs() should exclude the skipped func.
	funcs := p.funcs()
	if len(funcs) != 2 {
		t.Errorf("expected 2 funcs, got %d", len(funcs))
	}
	for _, fn := range funcs {
		if fn.name == "test-lint" {
			t.Error("expected test-lint to be excluded from funcs()")
		}
	}
}

func TestPaths_Skip_Multiple(t *testing.T) {
	fn1 := Func("test-format", "format code", func(_ context.Context) error { return nil })
	fn2 := Func("test-lint", "lint code", func(_ context.Context) error { return nil })
	fn3 := Func("test-build", "build code", func(_ context.Context) error { return nil })

	p := Paths(Serial(fn1, fn2, fn3)).In(".").Skip(fn2).Skip(fn3)

	funcs := p.funcs()
	if len(funcs) != 1 {
		t.Errorf("expected 1 func, got %d", len(funcs))
	}
	if funcs[0].name != "test-format" {
		t.Errorf("expected test-format, got %s", funcs[0].name)
	}
}

func TestPaths_Skip_Immutability(t *testing.T) {
	fn1 := Func("test-format", "format code", func(_ context.Context) error { return nil })
	fn2 := Func("test-lint", "lint code", func(_ context.Context) error { return nil })

	p1 := Paths(Serial(fn1, fn2)).In(".")
	p2 := p1.Skip(fn2)

	// p1 should still have both funcs.
	if len(p1.funcs()) != 2 {
		t.Error("p1 should still have 2 funcs (immutability violated)")
	}
	// p2 should have only 1 func.
	if len(p2.funcs()) != 1 {
		t.Error("p2 should have 1 func")
	}
}

func TestPaths_Skip_WithPaths_Immutability(t *testing.T) {
	fn1 := Func("fn1", "func 1", func(_ context.Context) error { return nil })

	p1 := Paths(fn1).In(".")
	p2 := p1.Skip(fn1, "docs")

	// p1 should not have any skip rules.
	if len(p1.skipRules) > 0 {
		t.Error("p1 should not have skipRules (immutability violated)")
	}
	// p2 should have skipRules with path-specific rule.
	if len(p2.skipRules) != 1 {
		t.Error("p2 should have 1 skipRule for fn1")
	}
	if len(p2.skipRules[0].paths) != 1 || p2.skipRules[0].paths[0] != "docs" {
		t.Error("p2 skipRule should have 'docs' path")
	}
}
