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

// TestTaskID_String tests the taskID string representation.
func TestTaskID_String(t *testing.T) {
	tests := []struct {
		name     string
		id       taskID
		expected string
	}{
		{
			name:     "SimpleTask",
			id:       taskID{Name: "build", Path: "."},
			expected: "build@.",
		},
		{
			name:     "TaskWithPath",
			id:       taskID{Name: "test", Path: "services/api"},
			expected: "test@services/api",
		},
		{
			name:     "TaskWithSuffix",
			id:       taskID{Name: "py-test:3.9", Path: "."},
			expected: "py-test:3.9@.",
		},
		{
			name:     "TaskWithSuffixAndPath",
			id:       taskID{Name: "py-test:3.9", Path: "pkg/lib"},
			expected: "py-test:3.9@pkg/lib",
		},
		{
			name:     "TaskWithNestedSuffix",
			id:       taskID{Name: "test:a:b", Path: "."},
			expected: "test:a:b@.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.String(); got != tt.expected {
				t.Errorf("taskID.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}
