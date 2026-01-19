package pk

import (
	"context"
	"sync/atomic"
	"testing"
)

func TestTask_Run_Deduplication(t *testing.T) {
	var runCount atomic.Int32

	task := NewTask("dedup-task", func(ctx context.Context, opts map[string]any) error {
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

	// Second run should be skipped (same task, global dedup).
	if err := task.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after second run (should skip), got %d", got)
	}

	// Run with different path context should STILL be skipped (global dedup).
	ctxServices := WithPath(ctx, "services")
	if err := task.run(ctxServices); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after run with different path (global dedup), got %d", got)
	}
}

func TestTask_Run_ForceRun(t *testing.T) {
	var runCount atomic.Int32

	task := NewTask("force-task", func(ctx context.Context, opts map[string]any) error {
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

	task := NewTask("no-tracker-task", func(ctx context.Context, opts map[string]any) error {
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

func TestTask_Run_DifferentTasksSameName(t *testing.T) {
	var runCount1, runCount2 atomic.Int32

	// Two different task instances with the same name.
	task1 := NewTask("same-name", func(ctx context.Context, opts map[string]any) error {
		runCount1.Add(1)
		return nil
	})
	task2 := NewTask("same-name", func(ctx context.Context, opts map[string]any) error {
		runCount2.Add(1)
		return nil
	})

	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// Both should run because deduplication is by pointer, not name.
	if err := task1.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := task2.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := runCount1.Load(); got != 1 {
		t.Errorf("expected task1 runCount=1, got %d", got)
	}
	if got := runCount2.Load(); got != 1 {
		t.Errorf("expected task2 runCount=1, got %d", got)
	}
}
