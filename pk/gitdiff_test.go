package pk

import (
	"context"
	"io"
	"testing"
)

func TestRunGitDiff_Disabled(t *testing.T) {
	ctx := context.Background()
	ctx = withGitDiffEnabled(ctx, false)
	ctx = WithOutput(ctx, &Output{Stdout: io.Discard, Stderr: io.Discard})

	// Should return nil immediately when git diff is disabled
	if err := runGitDiff(ctx); err != nil {
		t.Errorf("runGitDiff() with disabled flag returned error: %v", err)
	}
}

func TestGitDiffEnabledFromContext_Default(t *testing.T) {
	ctx := context.Background()

	// Default should be false (git diff disabled)
	if gitDiffEnabledFromContext(ctx) {
		t.Error("expected gitDiffEnabled to be false by default")
	}
}

func TestGitDiffEnabledFromContext_Enabled(t *testing.T) {
	ctx := context.Background()
	ctx = withGitDiffEnabled(ctx, true)

	if !gitDiffEnabledFromContext(ctx) {
		t.Error("expected gitDiffEnabled to be true after setting")
	}
}

func TestExecutionTracker_Executed(t *testing.T) {
	tracker := newExecutionTracker()
	tracker.markDone("task1", ".")
	tracker.markDone("task2", "foo")
	tracker.markDone("task1", "bar") // same task, different path

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
