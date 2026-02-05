package pk

import (
	"context"
	"flag"
	"sync/atomic"
	"testing"
)

func TestTask_Run_Deduplication(t *testing.T) {
	var runCount atomic.Int32

	task := NewTask(TaskConfig{Name: "dedup-task", Usage: "test task", Body: Do(func(_ context.Context) error {
		runCount.Add(1)
		return nil
	})})

	// Create context with tracker.
	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// First run should execute.
	if err := task.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after first run, got %d", got)
	}

	// Second run should be skipped (same task+path, deduplicated).
	if err := task.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after second run (should skip), got %d", got)
	}

	// Run with different path context should execute (different path).
	ctxServices := ContextWithPath(ctx, "services")
	if err := task.run(ctxServices); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after run with different path, got %d", got)
	}

	// Run again with services path should be skipped (same task+path).
	if err := task.run(ctxServices); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after second run with services path (should skip), got %d", got)
	}
}

func TestTask_Run_ForceRun(t *testing.T) {
	var runCount atomic.Int32

	task := NewTask(TaskConfig{Name: "force-task", Usage: "test task", Body: Do(func(_ context.Context) error {
		runCount.Add(1)
		return nil
	})})

	// Create context with tracker.
	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// First run should execute.
	if err := task.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after first run, got %d", got)
	}

	// Second run should be skipped.
	if err := task.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after second run (should skip), got %d", got)
	}

	// Run with forceRun should execute.
	ctxForce := withForceRun(ctx)
	if err := task.run(ctxForce); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after run with forceRun, got %d", got)
	}
}

func TestTask_Run_NoTracker(t *testing.T) {
	var runCount atomic.Int32

	task := NewTask(TaskConfig{Name: "no-tracker-task", Usage: "test task", Body: Do(func(_ context.Context) error {
		runCount.Add(1)
		return nil
	})})

	// Context without tracker - should always run (no deduplication).
	ctx := context.Background()

	if err := task.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := task.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 when no tracker present, got %d", got)
	}
}

func TestTask_Run_SameNameSamePathDeduplicated(t *testing.T) {
	var runCount1, runCount2 atomic.Int32

	// Two different task instances with the same name.
	task1 := NewTask(TaskConfig{Name: "same-name", Usage: "test task 1", Body: Do(func(_ context.Context) error {
		runCount1.Add(1)
		return nil
	})})
	task2 := NewTask(TaskConfig{Name: "same-name", Usage: "test task 2", Body: Do(func(_ context.Context) error {
		runCount2.Add(1)
		return nil
	})})

	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// First task should run.
	if err := task1.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second task with same name+path should be skipped (deduplication by name+path).
	if err := task2.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := runCount1.Load(); got != 1 {
		t.Errorf("expected task1 runCount=1, got %d", got)
	}
	if got := runCount2.Load(); got != 0 {
		t.Errorf("expected task2 runCount=0 (deduplicated), got %d", got)
	}
}

func TestTask_Accessors(t *testing.T) {
	task := NewTask(TaskConfig{Name: "my-task", Usage: "my usage", Body: Do(func(_ context.Context) error {
		return nil
	})})

	if got := task.Name(); got != "my-task" {
		t.Errorf("expected Name()=%q, got %q", "my-task", got)
	}
	if got := task.Usage(); got != "my usage" {
		t.Errorf("expected Usage()=%q, got %q", "my usage", got)
	}
	// Flags() always returns a non-nil FlagSet (for uniform flag.ErrHelp handling).
	// For tasks created without flags, verify the FlagSet exists but is empty.
	if got := task.Flags(); got == nil {
		t.Errorf("expected Flags() to return non-nil FlagSet, got nil")
	} else {
		var flagCount int
		got.VisitAll(func(*flag.Flag) { flagCount++ })
		if flagCount != 0 {
			t.Errorf("expected empty FlagSet for task without flags, got %d flags", flagCount)
		}
	}
	if got := task.IsHidden(); got {
		t.Errorf("expected IsHidden()=false, got %v", got)
	}
}

func TestTask_Hidden(t *testing.T) {
	task := NewTask(TaskConfig{Name: "visible-task", Usage: "test task", Body: Do(func(_ context.Context) error {
		return nil
	})})

	// Should not be hidden.
	if task.IsHidden() {
		t.Error("task should not be hidden")
	}

	hiddenTask := NewTask(TaskConfig{Name: "hidden-task", Usage: "test task", Hidden: true, Body: Do(func(_ context.Context) error {
		return nil
	})})

	// Should be hidden.
	if !hiddenTask.IsHidden() {
		t.Error("hidden task should be hidden")
	}
}

func TestTask_Run_FlagOverrides(t *testing.T) {
	var flagValue string
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.StringVar(&flagValue, "myflag", "default", "usage")

	task := NewTask(TaskConfig{Name: "flag-task", Usage: "test task", Flags: fs, Body: Do(func(_ context.Context) error {
		return nil
	})})

	// Helper to create a Plan with pre-computed flags for the task.
	planWithFlags := func(flags map[string]any) *Plan {
		return &Plan{
			taskIndex: map[string]*taskInstance{
				"flag-task": {
					task:  task,
					name:  "flag-task",
					flags: flags,
				},
			},
		}
	}

	t.Run("DefaultValue", func(t *testing.T) {
		// Reset flag to default before test.
		if err := fs.Set("myflag", "default"); err != nil {
			t.Fatal(err)
		}

		ctx := context.Background()
		if err := task.run(ctx); err != nil {
			t.Fatal(err)
		}
		if flagValue != "default" {
			t.Errorf("expected default value, got %q", flagValue)
		}
	})

	t.Run("WithOverride", func(t *testing.T) {
		// Reset flag to default before test.
		if err := fs.Set("myflag", "default"); err != nil {
			t.Fatal(err)
		}

		ctx := context.Background()
		// Create Plan with pre-computed flag override.
		plan := planWithFlags(map[string]any{"myflag": "overridden"})
		ctx = context.WithValue(ctx, planKey{}, plan)

		if err := task.run(ctx); err != nil {
			t.Fatal(err)
		}
		if flagValue != "overridden" {
			t.Errorf("expected overridden value, got %q", flagValue)
		}
	})

	t.Run("PreMergedOverrides", func(t *testing.T) {
		// Reset flag to default before test.
		if err := fs.Set("myflag", "default"); err != nil {
			t.Fatal(err)
		}

		ctx := context.Background()
		// In the new model, flags are pre-merged during planning.
		// The final value ("inner") is what gets stored in the Plan.
		plan := planWithFlags(map[string]any{"myflag": "inner"})
		ctx = context.WithValue(ctx, planKey{}, plan)

		if err := task.run(ctx); err != nil {
			t.Fatal(err)
		}
		if flagValue != "inner" {
			t.Errorf("expected pre-merged override value, got %q", flagValue)
		}
	})
}

// TestTask_Run_NameSuffixDeduplication tests that tasks with same base name
// but different suffixes (via WithNameSuffix) are NOT deduplicated.
// This is critical for multi-version testing (e.g., py-test:3.9, py-test:3.10).
func TestTask_Run_NameSuffixDeduplication(t *testing.T) {
	var runCount atomic.Int32

	task := NewTask(TaskConfig{Name: "py-test", Usage: "test task", Body: Do(func(_ context.Context) error {
		runCount.Add(1)
		return nil
	})})

	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// Run with suffix "3.9" - should execute.
	ctx39 := contextWithNameSuffix(ctx, "3.9")
	if err := task.run(ctx39); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after py-test:3.9, got %d", got)
	}

	// Run with suffix "3.10" - should also execute (different effective name).
	ctx310 := contextWithNameSuffix(ctx, "3.10")
	if err := task.run(ctx310); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after py-test:3.10, got %d", got)
	}

	// Run again with suffix "3.9" - should be skipped (duplicate).
	if err := task.run(ctx39); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after duplicate py-test:3.9, got %d", got)
	}

	// Run with suffix "3.11" - should execute.
	ctx311 := contextWithNameSuffix(ctx, "3.11")
	if err := task.run(ctx311); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 3 {
		t.Errorf("expected runCount=3 after py-test:3.11, got %d", got)
	}
}

// TestTask_Run_GlobalDeduplicationIgnoresSuffix tests that global tasks
// deduplicate by base name only, ignoring name suffix.
// This is critical for install tasks that should only run once.
func TestTask_Run_GlobalDeduplicationIgnoresSuffix(t *testing.T) {
	var runCount atomic.Int32

	// Global task - should deduplicate by base name only.
	task := NewTask(TaskConfig{Name: "install:uv", Usage: "install uv", Hidden: true, Global: true, Body: Do(func(_ context.Context) error {
		runCount.Add(1)
		return nil
	})})

	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// Run with suffix "3.9" - should execute.
	ctx39 := contextWithNameSuffix(ctx, "3.9")
	if err := task.run(ctx39); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after first run, got %d", got)
	}

	// Run with suffix "3.10" - should be skipped (global uses base name only).
	ctx310 := contextWithNameSuffix(ctx, "3.10")
	if err := task.run(ctx310); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after second run with different suffix (global should skip), got %d", got)
	}

	// Run with different path - should still be skipped (global ignores path too).
	ctx39Services := ContextWithPath(ctx39, "services")
	if err := task.run(ctx39Services); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after run with different path (global should skip), got %d", got)
	}
}

// TestTask_Run_NonGlobalWithSuffixAndPath tests that non-global tasks
// deduplicate by (effective name, path) tuple.
func TestTask_Run_NonGlobalWithSuffixAndPath(t *testing.T) {
	var runCount atomic.Int32

	task := NewTask(TaskConfig{Name: "test", Usage: "test task", Body: Do(func(_ context.Context) error {
		runCount.Add(1)
		return nil
	})})

	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// test:3.9 at root - should execute.
	ctx39 := contextWithNameSuffix(ctx, "3.9")
	if err := task.run(ctx39); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1, got %d", got)
	}

	// test:3.9 at services - should execute (different path).
	ctx39Services := ContextWithPath(ctx39, "services")
	if err := task.run(ctx39Services); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2, got %d", got)
	}

	// test:3.10 at services - should execute (different suffix).
	ctx310Services := ContextWithPath(contextWithNameSuffix(ctx, "3.10"), "services")
	if err := task.run(ctx310Services); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 3 {
		t.Errorf("expected runCount=3, got %d", got)
	}

	// test:3.9 at services again - should be skipped (duplicate).
	if err := task.run(ctx39Services); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 3 {
		t.Errorf("expected runCount=3 (duplicate should skip), got %d", got)
	}
}

// TestTask_Run_EffectiveName tests that the effective name (base name + suffix)
// is used correctly for task identification.
func TestTask_Run_EffectiveName(t *testing.T) {
	t.Run("NoSuffix", func(t *testing.T) {
		task := NewTask(TaskConfig{Name: "test", Usage: "test task", Body: Do(func(_ context.Context) error {
			return nil
		})})
		ctx := context.Background()
		// Effective name should be base name when no suffix.
		effectiveName := task.name
		if suffix := nameSuffixFromContext(ctx); suffix != "" {
			effectiveName = task.name + ":" + suffix
		}
		if effectiveName != "test" {
			t.Errorf("expected effectiveName=%q, got %q", "test", effectiveName)
		}
	})

	t.Run("WithSuffix", func(t *testing.T) {
		task := NewTask(TaskConfig{Name: "py-test", Usage: "test task", Body: Do(func(_ context.Context) error {
			return nil
		})})
		ctx := contextWithNameSuffix(context.Background(), "3.9")
		effectiveName := task.name
		if suffix := nameSuffixFromContext(ctx); suffix != "" {
			effectiveName = task.name + ":" + suffix
		}
		if effectiveName != "py-test:3.9" {
			t.Errorf("expected effectiveName=%q, got %q", "py-test:3.9", effectiveName)
		}
	})

	t.Run("NestedSuffix", func(t *testing.T) {
		task := NewTask(TaskConfig{Name: "test", Usage: "test task", Body: Do(func(_ context.Context) error {
			return nil
		})})
		ctx := contextWithNameSuffix(context.Background(), "a")
		ctx = contextWithNameSuffix(ctx, "b")
		effectiveName := task.name
		if suffix := nameSuffixFromContext(ctx); suffix != "" {
			effectiveName = task.name + ":" + suffix
		}
		if effectiveName != "test:a:b" {
			t.Errorf("expected effectiveName=%q, got %q", "test:a:b", effectiveName)
		}
	})
}
