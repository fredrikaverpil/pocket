package pk

import (
	"context"
	"slices"
	"sync/atomic"
	"testing"
)

func TestShimDirectories_OnlyIncludePaths(t *testing.T) {
	// Test that deriveModuleDirectories returns only root + include path patterns,
	// not every resolved subdirectory.

	// Simulate pathMappings where tasks have includePaths
	pathMappings := map[string]pathInfo{
		"lint": {
			includePaths:  []string{"pk", "internal"},
			resolvedPaths: []string{"pk", "internal", "internal/shim", "internal/scaffold"},
		},
	}

	dirs := deriveModuleDirectories(pathMappings)

	// Should contain exactly root, pk, and internal (from includePaths)
	expected := []string{".", "internal", "pk"}
	if len(dirs) != len(expected) {
		t.Fatalf("expected %d directories, got %d: %v", len(expected), len(dirs), dirs)
	}

	for _, exp := range expected {
		if !slices.Contains(dirs, exp) {
			t.Errorf("expected %q in directories, got %v", exp, dirs)
		}
	}

	// Should NOT contain subdirectories (those are resolvedPaths, not includePaths)
	unwanted := []string{"internal/shim", "internal/scaffold", "pk/testdata"}
	for _, uw := range unwanted {
		if slices.Contains(dirs, uw) {
			t.Errorf("unexpected directory %q in %v", uw, dirs)
		}
	}
}

func TestShimDirectories_RootAlwaysIncluded(t *testing.T) {
	// Even with no include paths, root should be included
	dirs := deriveModuleDirectories(map[string]pathInfo{})

	if len(dirs) != 1 {
		t.Fatalf("expected 1 directory (root), got %d: %v", len(dirs), dirs)
	}
	if dirs[0] != "." {
		t.Errorf("expected root (.), got %q", dirs[0])
	}
}

func TestShimDirectories_Sorted(t *testing.T) {
	pathMappings := map[string]pathInfo{
		"task1": {includePaths: []string{"zebra"}},
		"task2": {includePaths: []string{"alpha"}},
		"task3": {includePaths: []string{"beta"}},
	}

	dirs := deriveModuleDirectories(pathMappings)

	// Should be sorted (. < alpha < beta < zebra)
	expected := []string{".", "alpha", "beta", "zebra"}
	for i, exp := range expected {
		if dirs[i] != exp {
			t.Errorf("expected dirs[%d]=%q, got %q", i, exp, dirs[i])
		}
	}
}

func TestTaskRunsInPath_RootSeesAllTasks(t *testing.T) {
	p := &plan{
		pathMappings: map[string]pathInfo{
			"hello": {
				includePaths:  nil, // root-only task
				resolvedPaths: []string{"."},
			},
			"go-lint": {
				includePaths:  []string{"pk", "internal"},
				resolvedPaths: []string{"pk", "internal", "internal/shim"},
			},
		},
	}

	// From root, both tasks should be visible
	if !p.taskRunsInPath("hello", "") {
		t.Error("hello should be visible from root (empty string)")
	}
	if !p.taskRunsInPath("hello", ".") {
		t.Error("hello should be visible from root (.)")
	}
	if !p.taskRunsInPath("go-lint", "") {
		t.Error("go-lint should be visible from root")
	}
	if !p.taskRunsInPath("go-lint", ".") {
		t.Error("go-lint should be visible from root")
	}
}

func TestTaskRunsInPath_FilteredByIncludePath(t *testing.T) {
	p := &plan{
		pathMappings: map[string]pathInfo{
			"hello": {
				includePaths:  nil, // root-only task
				resolvedPaths: []string{"."},
			},
			"go-lint": {
				includePaths:  []string{"pk", "internal"},
				resolvedPaths: []string{"pk", "internal", "internal/shim"},
			},
		},
	}

	// From pk path
	if p.taskRunsInPath("hello", "pk") {
		t.Error("hello should NOT be visible from pk (it's root-only)")
	}
	if !p.taskRunsInPath("go-lint", "pk") {
		t.Error("go-lint should be visible from pk")
	}

	// From internal path
	if p.taskRunsInPath("hello", "internal") {
		t.Error("hello should NOT be visible from internal")
	}
	if !p.taskRunsInPath("go-lint", "internal") {
		t.Error("go-lint should be visible from internal")
	}

	// From unrelated path
	if p.taskRunsInPath("go-lint", "vendor") {
		t.Error("go-lint should NOT be visible from vendor")
	}
}

func TestTaskRunsInPath_UnknownTask(t *testing.T) {
	p := &plan{
		pathMappings: map[string]pathInfo{},
	}

	// Unknown task visible from root
	if !p.taskRunsInPath("unknown", "") {
		t.Error("unknown task should be visible from root")
	}

	// Unknown task not visible from subdirectory
	if p.taskRunsInPath("unknown", "pk") {
		t.Error("unknown task should NOT be visible from pk")
	}
}

func TestTaskExecution_ScopedToPathContext(t *testing.T) {
	var runCount atomic.Int32
	var executedPaths []string

	task := NewTask("scoped-task", "test task", nil, Do(func(ctx context.Context) error {
		runCount.Add(1)
		executedPaths = append(executedPaths, PathFromContext(ctx))
		return nil
	}))

	// Create plan with task mapped to multiple paths
	p := &plan{
		tasks: []*Task{task},
		pathMappings: map[string]pathInfo{
			"scoped-task": {
				includePaths:  []string{"pk", "internal"},
				resolvedPaths: []string{"pk", "internal"},
			},
		},
	}

	// Test 1: With TASK_SCOPE="pk", should only run in pk
	t.Run("scoped to pk", func(t *testing.T) {
		runCount.Store(0)
		executedPaths = nil

		t.Setenv("TASK_SCOPE", "pk")

		ctx := context.Background()
		ctx = withExecutionTracker(ctx, newExecutionTracker())

		if err := executeTask(ctx, task, p); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := runCount.Load(); got != 1 {
			t.Errorf("expected 1 execution, got %d", got)
		}
		if len(executedPaths) != 1 || executedPaths[0] != "pk" {
			t.Errorf("expected execution in [pk], got %v", executedPaths)
		}
	})

	// Test 2: Without TASK_SCOPE, should run in all paths
	t.Run("all paths", func(t *testing.T) {
		runCount.Store(0)
		executedPaths = nil

		t.Setenv("TASK_SCOPE", "")

		ctx := context.Background()
		ctx = withExecutionTracker(ctx, newExecutionTracker())

		if err := executeTask(ctx, task, p); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := runCount.Load(); got != 2 {
			t.Errorf("expected 2 executions, got %d", got)
		}
		if !slices.Contains(executedPaths, "pk") || !slices.Contains(executedPaths, "internal") {
			t.Errorf("expected execution in [pk, internal], got %v", executedPaths)
		}
	})

	// Test 3: TASK_SCOPE="." should behave like root (all paths)
	t.Run("root context explicit", func(t *testing.T) {
		runCount.Store(0)
		executedPaths = nil

		t.Setenv("TASK_SCOPE", ".")

		ctx := context.Background()
		ctx = withExecutionTracker(ctx, newExecutionTracker())

		if err := executeTask(ctx, task, p); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := runCount.Load(); got != 2 {
			t.Errorf("expected 2 executions with TASK_SCOPE=., got %d", got)
		}
	})
}
