package pk

import (
	"context"
	"sync"
	"testing"
)

func TestExecutionTracker_MarkDone(t *testing.T) {
	tracker := newExecutionTracker()

	task1 := NewTask("task1", nil)
	task2 := NewTask("task2", nil)

	// First call should return false (not already done).
	if alreadyDone := tracker.markDone(task1); alreadyDone {
		t.Error("expected first markDone to return false")
	}

	// Second call with same task should return true (global dedup).
	if alreadyDone := tracker.markDone(task1); !alreadyDone {
		t.Error("expected second markDone with same task to return true")
	}

	// Different task should return false.
	if alreadyDone := tracker.markDone(task2); alreadyDone {
		t.Error("expected markDone with different task to return false")
	}
}

func TestExecutionTracker_Concurrent(t *testing.T) {
	tracker := newExecutionTracker()
	task := NewTask("concurrent-task", nil)

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Count how many goroutines think they're first.
	var firstCount int
	var mu sync.Mutex

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if !tracker.markDone(task) {
				mu.Lock()
				firstCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Exactly one goroutine should have been first.
	if firstCount != 1 {
		t.Errorf("expected exactly 1 goroutine to be first, got %d", firstCount)
	}
}

func TestExecutionTrackerContext(t *testing.T) {
	ctx := context.Background()

	// No tracker set, should return nil.
	if tracker := executionTrackerFromContext(ctx); tracker != nil {
		t.Error("expected nil tracker from empty context")
	}

	// Set tracker and retrieve it.
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	retrieved := executionTrackerFromContext(ctx)
	if retrieved != tracker {
		t.Error("expected to retrieve the same tracker from context")
	}
}

func TestForceRunContext(t *testing.T) {
	ctx := context.Background()

	// Default should be false.
	if forceRunFromContext(ctx) {
		t.Error("expected forceRun to be false by default")
	}

	// Set forceRun and check.
	ctx = withForceRun(ctx)
	if !forceRunFromContext(ctx) {
		t.Error("expected forceRun to be true after withForceRun")
	}
}
