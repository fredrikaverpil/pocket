package pocket

import (
	"context"
	"testing"
)

// mockRunnable is a minimal Runnable for testing.
type mockRunnable struct {
	tasks []*Task
}

func (m *mockRunnable) Run(_ context.Context) error {
	return nil
}

func (m *mockRunnable) Tasks() []*Task {
	return m.tasks
}

// mockDetectable is a Runnable that implements Detectable.
type mockDetectable struct {
	mockRunnable
	detectFn func() []string
}

func (m *mockDetectable) DefaultDetect() func() []string {
	return m.detectFn
}

func TestPaths_In(t *testing.T) {
	r := &mockRunnable{}
	p := Paths(r).In("proj1", "proj2")

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
	r := &mockDetectable{
		detectFn: func() []string {
			return []string{"proj1", "proj2", "vendor"}
		},
	}
	p := Paths(r).Detect().Except("vendor")

	resolved := p.Resolve()
	if len(resolved) != 2 {
		t.Errorf("expected 2 paths, got %d: %v", len(resolved), resolved)
	}
	if p.RunsIn("vendor") {
		t.Error("expected vendor to be excluded")
	}
}

func TestPaths_DetectBy(t *testing.T) {
	r := &mockRunnable{}
	p := Paths(r).DetectBy(func() []string {
		return []string{"a", "b", "c"}
	})

	resolved := p.Resolve()
	if len(resolved) != 3 {
		t.Errorf("expected 3 paths, got %d: %v", len(resolved), resolved)
	}
}

func TestPaths_Detect_WithDetectable(t *testing.T) {
	r := &mockDetectable{
		detectFn: func() []string {
			return []string{"mod1", "mod2"}
		},
	}
	p := Paths(r).Detect()

	resolved := p.Resolve()
	if len(resolved) != 2 {
		t.Errorf("expected 2 paths, got %d: %v", len(resolved), resolved)
	}
}

func TestPaths_Detect_WithoutDetectable(t *testing.T) {
	r := &mockRunnable{} // doesn't implement Detectable
	p := Paths(r).Detect()

	resolved := p.Resolve()
	if len(resolved) != 0 {
		t.Errorf("expected 0 paths, got %d: %v", len(resolved), resolved)
	}
}

func TestPaths_CombineDetectAndInclude(t *testing.T) {
	r := &mockDetectable{
		detectFn: func() []string {
			return []string{"detected1", "detected2"}
		},
	}
	p := Paths(r).Detect().In("detected1") // filter to only detected1

	resolved := p.Resolve()
	if len(resolved) != 1 {
		t.Errorf("expected 1 path, got %d: %v", len(resolved), resolved)
	}
	if !p.RunsIn("detected1") {
		t.Error("expected RunsIn(detected1) to be true")
	}
}

func TestPaths_Immutability(t *testing.T) {
	r := &mockRunnable{}
	p1 := Paths(r).In("proj1")
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

func TestPaths_Tasks(t *testing.T) {
	task1 := &Task{Name: "test-task"}
	r := &mockRunnable{tasks: []*Task{task1}}
	p := Paths(r).In(".")

	tasks := p.Tasks()
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Name != "test-task" {
		t.Errorf("expected task name 'test-task', got %s", tasks[0].Name)
	}
}

func TestPaths_RegexPatterns(t *testing.T) {
	r := &mockDetectable{
		detectFn: func() []string {
			return []string{"services/api", "services/web", "tools/cli"}
		},
	}
	p := Paths(r).Detect().In("services/.*")

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
	r := &mockRunnable{}
	p := Paths(r).In(".")

	if !p.RunsIn(".") {
		t.Error("expected RunsIn(.) to be true")
	}
	if p.RunsIn("subdir") {
		t.Error("expected RunsIn(subdir) to be false")
	}
}

func TestCollectPathMappings(t *testing.T) {
	task1 := &Task{Name: "task1"}
	task2 := &Task{Name: "task2"}
	task3 := &Task{Name: "task3"}

	// Create a mock Detectable that returns specific directories.
	detectable := &mockDetectable{
		mockRunnable: mockRunnable{tasks: []*Task{task1, task2}},
		detectFn:     func() []string { return []string{"proj1", "proj2"} },
	}

	// Wrap with Paths().Detect().
	wrapped := Paths(detectable).Detect()

	// Create a serial runnable with wrapped and unwrapped tasks.
	runnable := Serial(wrapped, task3)

	mappings := CollectPathMappings(runnable)

	// task1 and task2 should be in mappings (from wrapped Paths).
	if _, ok := mappings["task1"]; !ok {
		t.Error("expected task1 to be in mappings")
	}
	if _, ok := mappings["task2"]; !ok {
		t.Error("expected task2 to be in mappings")
	}
	// task3 should NOT be in mappings (not wrapped with Paths).
	if _, ok := mappings["task3"]; ok {
		t.Error("expected task3 to NOT be in mappings")
	}

	// Check that the paths resolve correctly.
	if paths, ok := mappings["task1"]; ok {
		resolved := paths.Resolve()
		if len(resolved) != 2 {
			t.Errorf("expected 2 resolved paths, got %d", len(resolved))
		}
	}
}

func TestCollectModuleDirectories(t *testing.T) {
	detectable := &mockDetectable{
		detectFn: func() []string { return []string{"proj1", "proj2"} },
	}
	wrapped := Paths(detectable).Detect()

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
