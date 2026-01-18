package pocket

import (
	"context"
	"testing"
)

func TestRunIn_Include(t *testing.T) {
	fn := Task("test", "test", func(_ context.Context) error { return nil })
	p := RunIn(fn, Include("proj1", "proj2"))

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

func TestRunIn_Exclude(t *testing.T) {
	fn := Task("test", "test", func(_ context.Context) error { return nil })
	p := RunIn(fn,
		Detect(func() []string {
			return []string{"proj1", "proj2", "vendor"}
		}),
		Exclude("vendor"),
	)

	resolved := p.Resolve()
	if len(resolved) != 2 {
		t.Errorf("expected 2 paths, got %d: %v", len(resolved), resolved)
	}
	if p.RunsIn("vendor") {
		t.Error("expected vendor to be excluded")
	}
}

func TestRunIn_Detect(t *testing.T) {
	fn := Task("test", "test", func(_ context.Context) error { return nil })
	p := RunIn(fn, Detect(func() []string {
		return []string{"a", "b", "c"}
	}))

	resolved := p.Resolve()
	if len(resolved) != 3 {
		t.Errorf("expected 3 paths, got %d: %v", len(resolved), resolved)
	}
}

func TestRunIn_Detect_ReturnsCorrectPaths(t *testing.T) {
	fn := Task("test", "test", func(_ context.Context) error { return nil })
	p := RunIn(fn, Detect(func() []string {
		return []string{"mod1", "mod2"}
	}))

	resolved := p.Resolve()
	if len(resolved) != 2 {
		t.Errorf("expected 2 paths, got %d: %v", len(resolved), resolved)
	}
}

func TestRunIn_NoDetect_ReturnsEmpty(t *testing.T) {
	fn := Task("test", "test", func(_ context.Context) error { return nil })
	p := RunIn(fn) // no detection set

	resolved := p.Resolve()
	if len(resolved) != 0 {
		t.Errorf("expected 0 paths, got %d: %v", len(resolved), resolved)
	}
}

func TestRunIn_CombineDetectAndInclude(t *testing.T) {
	fn := Task("test", "test", func(_ context.Context) error { return nil })
	p := RunIn(fn,
		Detect(func() []string {
			return []string{"detected1", "detected2"}
		}),
		Include("detected1"), // filter to only detected1
	)

	resolved := p.Resolve()
	if len(resolved) != 1 {
		t.Errorf("expected 1 path, got %d: %v", len(resolved), resolved)
	}
	if !p.RunsIn("detected1") {
		t.Error("expected RunsIn(detected1) to be true")
	}
}

func TestRunIn_TaskDefs(t *testing.T) {
	fn := Task("test-func", "test func", func(_ context.Context) error { return nil })
	p := RunIn(fn, Include("."))

	engine := NewEngine(p)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		t.Fatalf("Engine.Plan() failed: %v", err)
	}
	funcs := plan.TaskDefs()
	if len(funcs) != 1 {
		t.Errorf("expected 1 func, got %d", len(funcs))
	}
	if funcs[0].name != "test-func" {
		t.Errorf("expected func name 'test-func', got %s", funcs[0].name)
	}
}

func TestRunIn_RegexPatterns(t *testing.T) {
	fn := Task("test", "test", func(_ context.Context) error { return nil })
	p := RunIn(fn,
		Detect(func() []string {
			return []string{"services/api", "services/web", "tools/cli"}
		}),
		Include("services/.*"),
	)

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

func TestRunIn_RootOnly(t *testing.T) {
	fn := Task("test", "test", func(_ context.Context) error { return nil })
	p := RunIn(fn, Include("."))

	if !p.RunsIn(".") {
		t.Error("expected RunsIn(.) to be true")
	}
	if p.RunsIn("subdir") {
		t.Error("expected RunsIn(subdir) to be false")
	}
}

func TestEngine_PathMappings(t *testing.T) {
	fn1 := Task("fn1", "func 1", func(_ context.Context) error { return nil })
	fn2 := Task("fn2", "func 2", func(_ context.Context) error { return nil })
	fn3 := Task("fn3", "func 3", func(_ context.Context) error { return nil })

	// Wrap fn1 and fn2 with RunIn and Detect.
	wrapped := RunIn(Serial(fn1, fn2), Detect(func() []string {
		return []string{"proj1", "proj2"}
	}))

	// Create a serial runnable with wrapped and unwrapped funcs.
	runnable := Serial(wrapped, fn3)

	// Use Engine.Plan() to collect path mappings
	engine := NewEngine(runnable)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan() failed: %v", err)
	}
	mappings := plan.PathMappings()

	// fn1 and fn2 should be in mappings (from wrapped RunIn).
	if _, ok := mappings["fn1"]; !ok {
		t.Error("expected fn1 to be in mappings")
	}
	if _, ok := mappings["fn2"]; !ok {
		t.Error("expected fn2 to be in mappings")
	}
	// fn3 should NOT be in mappings (not wrapped with RunIn).
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

func TestModuleDirectoriesFromPlan(t *testing.T) {
	fn := Task("test", "test", func(_ context.Context) error { return nil })
	wrapped := RunIn(fn, Detect(func() []string {
		return []string{"proj1", "proj2"}
	}))

	// Use Engine.Plan() to collect module directories (consolidated approach)
	engine := NewEngine(wrapped)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		t.Fatalf("Engine.Plan() failed: %v", err)
	}
	dirs := plan.ModuleDirectories()

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
