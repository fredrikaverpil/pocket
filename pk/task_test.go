package pk

import (
	"context"
	"sync/atomic"
	"testing"
)

func TestTask_Run_Deduplication(t *testing.T) {
	var runCount atomic.Int32

	task := NewTask("dedup-task", "test task", func(ctx context.Context) error {
		runCount.Add(1)
		return nil
	})

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
	ctxServices := WithPath(ctx, "services")
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

	task := NewTask("force-task", "test task", func(ctx context.Context) error {
		runCount.Add(1)
		return nil
	})

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

	task := NewTask("no-tracker-task", "test task", func(ctx context.Context) error {
		runCount.Add(1)
		return nil
	})

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
	task1 := NewTask("same-name", "test task 1", func(ctx context.Context) error {
		runCount1.Add(1)
		return nil
	})
	task2 := NewTask("same-name", "test task 2", func(ctx context.Context) error {
		runCount2.Add(1)
		return nil
	})

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
	task := NewTask("my-task", "my usage", func(ctx context.Context) error {
		return nil
	})

	if got := task.Name(); got != "my-task" {
		t.Errorf("expected Name()=%q, got %q", "my-task", got)
	}
	if got := task.Usage(); got != "my usage" {
		t.Errorf("expected Usage()=%q, got %q", "my usage", got)
	}
	if got := task.Flags(); got != nil {
		t.Errorf("expected Flags()=nil for task without flags, got %v", got)
	}
	if got := task.IsHidden(); got {
		t.Errorf("expected IsHidden()=false, got %v", got)
	}
}

func TestTask_Hidden(t *testing.T) {
	task := NewTask("visible-task", "test task", func(ctx context.Context) error {
		return nil
	})

	hiddenTask := task.Hidden()

	// Original should not be hidden.
	if task.IsHidden() {
		t.Error("original task should not be hidden")
	}

	// Hidden copy should be hidden.
	if !hiddenTask.IsHidden() {
		t.Error("hidden task should be hidden")
	}

	// Hidden copy should preserve other fields.
	if hiddenTask.Name() != task.Name() {
		t.Error("hidden task should have same name")
	}
	if hiddenTask.Usage() != task.Usage() {
		t.Error("hidden task should have same usage")
	}
}
