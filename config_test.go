package pocket

import (
	"context"
	"testing"
)

func TestConfig_WithDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       Config
		wantShimName string
		wantPosix    bool
	}{
		{
			name:         "empty config gets default shim name and posix",
			config:       Config{},
			wantShimName: "pok",
			wantPosix:    true,
		},
		{
			name: "custom shim name is preserved",
			config: Config{
				Shim: &ShimConfig{Name: "build", Posix: true},
			},
			wantShimName: "build",
			wantPosix:    true,
		},
		{
			name: "empty shim name gets default",
			config: Config{
				Shim: &ShimConfig{Posix: true},
			},
			wantShimName: "pok",
			wantPosix:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.config.WithDefaults()

			if got.Shim == nil {
				t.Fatal("WithDefaults().Shim is nil")
			}
			if got.Shim.Name != tt.wantShimName {
				t.Errorf("WithDefaults().Shim.Name = %q, want %q", got.Shim.Name, tt.wantShimName)
			}
		})
	}
}

func TestSerial_Tasks(t *testing.T) {
	t.Parallel()

	task1 := &Task{
		Name:  "test-format",
		Usage: "format test files",
		Action: func(_ context.Context, _ *RunContext) error {
			return nil
		},
	}
	task2 := &Task{
		Name:  "test-lint",
		Usage: "lint test files",
		Action: func(_ context.Context, _ *RunContext) error {
			return nil
		},
	}

	runnable := Serial(task1, task2)

	// Check tasks returns both tasks.
	tasks := runnable.Tasks()
	if len(tasks) != 2 {
		t.Errorf("Tasks() length = %d, want 2", len(tasks))
	}
}

func TestParallel_Tasks(t *testing.T) {
	t.Parallel()

	task1 := &Task{Name: "task1", Usage: "task 1"}
	task2 := &Task{Name: "task2", Usage: "task 2"}

	runnable := Parallel(task1, task2)

	tasks := runnable.Tasks()
	if len(tasks) != 2 {
		t.Errorf("Tasks() length = %d, want 2", len(tasks))
	}
}

func TestConfig_Run(t *testing.T) {
	t.Parallel()

	task1 := &Task{Name: "deploy", Usage: "deploy app"}
	task2 := &Task{Name: "release", Usage: "release app"}

	cfg := Config{
		AutoRun: Serial(task1, task2),
	}

	tasks := cfg.AutoRun.Tasks()
	if len(tasks) != 2 {
		t.Errorf("Run.Tasks() length = %d, want 2", len(tasks))
	}
}

func TestNested_Serial_Parallel(t *testing.T) {
	t.Parallel()

	task1 := &Task{Name: "task1", Usage: "task 1"}
	task2 := &Task{Name: "task2", Usage: "task 2"}
	task3 := &Task{Name: "task3", Usage: "task 3"}

	runnable := Serial(
		task1,
		Parallel(task2, task3),
	)

	tasks := runnable.Tasks()
	if len(tasks) != 3 {
		t.Errorf("Tasks() length = %d, want 3", len(tasks))
	}
}
