package pk

import (
	"context"
	"sync/atomic"
	"testing"
)

func TestWithForceRun(t *testing.T) {
	var runCount atomic.Int32

	task := NewTask("path-task", "test task", nil, Do(func(_ context.Context) error {
		runCount.Add(1)
		return nil
	}))

	// Create context with tracker.
	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// Create pathFilter without forceRun.
	pf := WithOptions(task).(*pathFilter)
	pf.resolvedPaths = []string{"."} // Simulate resolved paths.

	// First run should execute.
	if err := pf.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after first run, got %d", got)
	}

	// Second run should be skipped (same task+path).
	if err := pf.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after second run (should skip), got %d", got)
	}

	// Create pathFilter with forceRun.
	pfForce := WithOptions(task, WithForceRun()).(*pathFilter)
	pfForce.resolvedPaths = []string{"."}

	// Should execute despite already having run.
	if err := pfForce.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after run with WithForceRun, got %d", got)
	}
}

func TestPathFilter_MultiplePaths(t *testing.T) {
	var paths []string

	task := NewTask("multi-path-task", "test task", nil, Do(func(ctx context.Context) error {
		paths = append(paths, PathFromContext(ctx))
		return nil
	}))

	// Context WITHOUT tracker - task runs for each path (no dedup).
	ctx := context.Background()

	// Create pathFilter with multiple resolved paths.
	pf := WithOptions(task).(*pathFilter)
	pf.resolvedPaths = []string{"services/api", "services/web", "pkg/utils"}

	if err := pf.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Without tracker, should run once per path.
	if len(paths) != 3 {
		t.Errorf("expected 3 paths, got %d: %v", len(paths), paths)
	}

	expected := []string{"services/api", "services/web", "pkg/utils"}
	for i, p := range paths {
		if p != expected[i] {
			t.Errorf("expected path[%d]=%q, got %q", i, expected[i], p)
		}
	}
}

func TestPathFilter_MultiplePathsWithDedup(t *testing.T) {
	var runCount atomic.Int32

	task := NewTask("multi-path-dedup-task", "test task", nil, Do(func(_ context.Context) error {
		runCount.Add(1)
		return nil
	}))

	// Context WITH tracker - dedup by (task, path) means each path runs once.
	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	pf := WithOptions(task).(*pathFilter)
	pf.resolvedPaths = []string{"services/api", "services/web", "pkg/utils"}

	if err := pf.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With (task, path) dedup, task runs once per unique path.
	if got := runCount.Load(); got != 3 {
		t.Errorf("expected runCount=3 (once per path), got %d", got)
	}

	// Running again should not add more executions (all paths already done).
	if err := pf.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 3 {
		t.Errorf("expected runCount=3 after second run (all deduplicated), got %d", got)
	}
}

func TestPathFilter_DeduplicationByTaskAndPath(t *testing.T) {
	var runCount atomic.Int32

	task := NewTask("path-dedup-task", "test task", nil, Do(func(_ context.Context) error {
		runCount.Add(1)
		return nil
	}))

	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// First pathFilter runs in services/api.
	pf1 := WithOptions(task).(*pathFilter)
	pf1.resolvedPaths = []string{"services/api"}

	// Second pathFilter runs in a different path - should run (different path).
	pf2 := WithOptions(task).(*pathFilter)
	pf2.resolvedPaths = []string{"services/web"}

	if err := pf1.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after pf1, got %d", got)
	}

	// Same task at different path should run (dedup by task+path).
	if err := pf2.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after pf2 (different path), got %d", got)
	}

	// Running pf1 again should skip (same task+path).
	if err := pf1.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after pf1 again (deduplicated), got %d", got)
	}
}
