package pocket

import (
	"context"
	"testing"
)

// TestOptions is a typed options struct for testing.
type TestOptions struct {
	Name  string `arg:"name"  usage:"who to greet"`
	Count int    `arg:"count" usage:"number of times"`
	Debug bool   `arg:"debug" usage:"enable debug mode"`
}

func TestTask_TypedArgs_Defaults(t *testing.T) {
	var received TestOptions

	task := &Task{
		Name:    "test-task",
		Options: TestOptions{Name: "world", Count: 10, Debug: false},
		Action: func(_ context.Context, tc *TaskContext) error {
			received = GetOptions[TestOptions](tc)
			return nil
		},
	}

	// Run without any CLI args - should get defaults.
	exec := NewExecution(StdOutput(), false, ".")
	if err := task.Run(context.Background(), exec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.Name != "world" {
		t.Errorf("expected Name='world', got %q", received.Name)
	}
	if received.Count != 10 {
		t.Errorf("expected Count=10, got %d", received.Count)
	}
	if received.Debug {
		t.Error("expected Debug=false")
	}
}

func TestTask_TypedArgs_CLIOverride(t *testing.T) {
	var received TestOptions

	task := &Task{
		Name:    "test-task",
		Options: TestOptions{Name: "world", Count: 10, Debug: false},
		Action: func(_ context.Context, tc *TaskContext) error {
			received = GetOptions[TestOptions](tc)
			return nil
		},
	}

	// Override via CLI args set on Execution.
	exec := NewExecution(StdOutput(), false, ".")
	exec.SetTaskArgs(task.Name, map[string]string{
		"name":  "Freddy",
		"count": "42",
		"debug": "true",
	})

	if err := task.Run(context.Background(), exec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.Name != "Freddy" {
		t.Errorf("expected Name='Freddy', got %q", received.Name)
	}
	if received.Count != 42 {
		t.Errorf("expected Count=42, got %d", received.Count)
	}
	if !received.Debug {
		t.Error("expected Debug=true")
	}
}

func TestTask_TypedArgs_PartialOverride(t *testing.T) {
	var received TestOptions

	task := &Task{
		Name:    "test-task",
		Options: TestOptions{Name: "world", Count: 10, Debug: false},
		Action: func(_ context.Context, tc *TaskContext) error {
			received = GetOptions[TestOptions](tc)
			return nil
		},
	}

	// Override only one field.
	exec := NewExecution(StdOutput(), false, ".")
	exec.SetTaskArgs(task.Name, map[string]string{
		"name": "partial",
	})

	if err := task.Run(context.Background(), exec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.Name != "partial" {
		t.Errorf("expected Name='partial', got %q", received.Name)
	}
	if received.Count != 10 {
		t.Errorf("expected Count=10 (default), got %d", received.Count)
	}
}

func TestTask_NoArgs(t *testing.T) {
	ran := false

	task := &Task{
		Name: "test-task",
		// No Args field set
		Action: func(_ context.Context, tc *TaskContext) error {
			ran = true
			// GetArgs on nil should return zero value.
			args := GetOptions[TestOptions](tc)
			if args.Name != "" {
				t.Errorf("expected empty Name, got %q", args.Name)
			}
			return nil
		},
	}

	exec := NewExecution(StdOutput(), false, ".")
	if err := task.Run(context.Background(), exec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ran {
		t.Error("action should have run")
	}
}

func TestTask_ActionReceivesVerbose(t *testing.T) {
	var receivedVerbose bool

	task := &Task{
		Name: "test-task",
		Action: func(_ context.Context, tc *TaskContext) error {
			receivedVerbose = tc.Verbose
			return nil
		},
	}

	// Run without verbose.
	ctx := context.Background()
	exec := NewExecution(StdOutput(), false, ".")
	if err := task.Run(ctx, exec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedVerbose {
		t.Error("expected Verbose=false")
	}

	// Run with verbose (new task since execution tracking is per-Execution).
	task2 := &Task{
		Name: "test-task-verbose",
		Action: func(_ context.Context, tc *TaskContext) error {
			receivedVerbose = tc.Verbose
			return nil
		},
	}
	execVerbose := NewExecution(StdOutput(), true, ".")
	if err := task2.Run(ctx, execVerbose); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !receivedVerbose {
		t.Error("expected Verbose=true")
	}
}
