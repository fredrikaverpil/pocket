package pk

import (
	"context"
	"sync"
	"testing"
)

func TestExecutionTracker_MarkDone(t *testing.T) {
	tracker := newExecutionTracker()

	// First call should return false (not already done).
	if alreadyDone := tracker.markDone(taskID{Name: "task1", Path: "path/a"}); alreadyDone {
		t.Error("expected first markDone to return false")
	}

	// Second call with same task+path should return true (deduplicated).
	if alreadyDone := tracker.markDone(taskID{Name: "task1", Path: "path/a"}); !alreadyDone {
		t.Error("expected second markDone with same task+path to return true")
	}

	// Same task, different path should return false (runs again).
	if alreadyDone := tracker.markDone(taskID{Name: "task1", Path: "path/b"}); alreadyDone {
		t.Error("expected markDone with same task but different path to return false")
	}

	// Different task, same path should return false.
	if alreadyDone := tracker.markDone(taskID{Name: "task2", Path: "path/a"}); alreadyDone {
		t.Error("expected markDone with different task to return false")
	}
}

func TestExecutionTracker_Concurrent(t *testing.T) {
	tracker := newExecutionTracker()

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Count how many goroutines think they're first.
	var firstCount int
	var mu sync.Mutex

	for range goroutines {
		go func() {
			defer wg.Done()
			if !tracker.markDone(taskID{Name: "concurrent-task", Path: "same/path"}) {
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

func TestExecutionTracker_Executed(t *testing.T) {
	tracker := newExecutionTracker()
	tracker.markDone(taskID{Name: "task1", Path: "."})
	tracker.markDone(taskID{Name: "task2", Path: "foo"})
	tracker.markDone(taskID{Name: "task1", Path: "bar"}) // same task, different path

	executed := tracker.executed()
	if len(executed) != 3 {
		t.Fatalf("expected 3 executions, got %d", len(executed))
	}

	// Check that all expected executions are present
	expected := map[string]bool{
		"task1:.":   true,
		"task2:foo": true,
		"task1:bar": true,
	}

	for _, exec := range executed {
		key := exec.TaskName + ":" + exec.Path
		if !expected[key] {
			t.Errorf("unexpected execution: %s", key)
		}
		delete(expected, key)
	}

	if len(expected) > 0 {
		t.Errorf("missing executions: %v", expected)
	}
}

func TestExecutionTracker_MarkWarning(t *testing.T) {
	tracker := newExecutionTracker()

	// Default: no warnings.
	if tracker.warnings() {
		t.Error("expected no warnings initially")
	}

	tracker.markWarning()

	if !tracker.warnings() {
		t.Error("expected warnings after markWarning")
	}
}

func TestExecutionTracker_MarkWarning_Concurrent(t *testing.T) {
	tracker := newExecutionTracker()

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			tracker.markWarning()
		})
	}
	wg.Wait()

	if !tracker.warnings() {
		t.Error("expected warnings after concurrent marking")
	}
}

// TestExecutionTracker_ExecutedWithSuffixes tests that executed() correctly
// parses task names that contain colons (e.g., "py-test:3.9").
func TestExecutionTracker_ExecutedWithSuffixes(t *testing.T) {
	tracker := newExecutionTracker()
	tracker.markDone(taskID{Name: "py-test:3.9", Path: "."})
	tracker.markDone(taskID{Name: "py-test:3.10", Path: "."})
	tracker.markDone(taskID{Name: "py-test:3.9", Path: "services"}) // same suffix, different path

	executed := tracker.executed()
	if len(executed) != 3 {
		t.Fatalf("expected 3 executions, got %d", len(executed))
	}

	// Check that all expected executions are present with correct parsing
	expected := map[string]bool{
		"py-test:3.9@.":        true,
		"py-test:3.10@.":       true,
		"py-test:3.9@services": true,
	}

	for _, exec := range executed {
		key := exec.TaskName + "@" + exec.Path
		if !expected[key] {
			t.Errorf("unexpected execution: %s (TaskName=%q, Path=%q)", key, exec.TaskName, exec.Path)
		}
		delete(expected, key)
	}

	if len(expected) > 0 {
		t.Errorf("missing executions: %v", expected)
	}
}
