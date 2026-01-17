package pocket

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCollectTasks_Simple(t *testing.T) {
	fn := Task("test", "test function", func(_ context.Context) error {
		return nil
	})

	tasks := CollectTasks(fn)

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	if tasks[0].Name != "test" {
		t.Errorf("expected name 'test', got %q", tasks[0].Name)
	}
	if tasks[0].Usage != "test function" {
		t.Errorf("expected usage 'test function', got %q", tasks[0].Usage)
	}
	if len(tasks[0].Paths) != 1 || tasks[0].Paths[0] != "." {
		t.Errorf("expected paths ['.'], got %v", tasks[0].Paths)
	}
	if tasks[0].Hidden {
		t.Error("expected task to not be hidden")
	}
}

func TestCollectTasks_Hidden(t *testing.T) {
	fn := Task("install:tool", "install tool", func(_ context.Context) error {
		return nil
	}, AsHidden())

	tasks := CollectTasks(fn)

	// Hidden tasks are included with Hidden=true (for CI filtering)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if !tasks[0].Hidden {
		t.Error("expected task to be marked as hidden")
	}
	if tasks[0].Name != "install:tool" {
		t.Errorf("expected name 'install:tool', got %q", tasks[0].Name)
	}
}

func TestCollectTasks_Serial(t *testing.T) {
	fn1 := Task("format", "format code", func(_ context.Context) error { return nil })
	fn2 := Task("lint", "lint code", func(_ context.Context) error { return nil })
	fn3 := Task("test", "run tests", func(_ context.Context) error { return nil })

	root := Serial(fn1, fn2, fn3)
	tasks := CollectTasks(root)

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	names := make([]string, len(tasks))
	for i, task := range tasks {
		names[i] = task.Name
	}

	expected := []string{"format", "lint", "test"}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("expected task %d to be %q, got %q", i, name, names[i])
		}
	}
}

func TestCollectTasks_Parallel(t *testing.T) {
	fn1 := Task("lint", "lint code", func(_ context.Context) error { return nil })
	fn2 := Task("test", "run tests", func(_ context.Context) error { return nil })

	root := Parallel(fn1, fn2)
	tasks := CollectTasks(root)

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestCollectTasks_WithNestedDeps(t *testing.T) {
	// Hidden install task
	install := Task("install:tool", "install", func(_ context.Context) error {
		return nil
	}, AsHidden())

	// Visible task with hidden dependency
	lint := Task("lint", "lint code", Serial(
		install,
		func(_ context.Context) error { return nil },
	))

	tasks := CollectTasks(lint)

	// Both tasks are included (hidden with Hidden=true)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	// First should be lint (visible)
	if tasks[0].Name != "lint" {
		t.Errorf("expected first task 'lint', got %q", tasks[0].Name)
	}
	if tasks[0].Hidden {
		t.Error("lint should not be hidden")
	}

	// Second should be install:tool (hidden)
	if tasks[1].Name != "install:tool" {
		t.Errorf("expected second task 'install:tool', got %q", tasks[1].Name)
	}
	if !tasks[1].Hidden {
		t.Error("install:tool should be hidden")
	}
}

func TestCollectTasks_Nil(t *testing.T) {
	tasks := CollectTasks(nil)

	if tasks != nil {
		t.Errorf("expected nil for nil input, got %v", tasks)
	}
}

func TestExportJSON_Simple(t *testing.T) {
	fn := Task("build", "build project", func(_ context.Context) error {
		return nil
	})

	data, err := ExportJSON(fn)
	if err != nil {
		t.Fatalf("ExportJSON() failed: %v", err)
	}

	// Verify it's valid JSON
	var tasks []TaskInfo
	if err := json.Unmarshal(data, &tasks); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Name != "build" {
		t.Errorf("expected name 'build', got %q", tasks[0].Name)
	}
}

func TestExportJSON_Empty(t *testing.T) {
	data, err := ExportJSON(nil)
	if err != nil {
		t.Fatalf("ExportJSON() failed: %v", err)
	}

	// Should be valid JSON (null or empty array)
	var tasks []TaskInfo
	if err := json.Unmarshal(data, &tasks); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestExportJSON_MultipleTasks(t *testing.T) {
	root := Serial(
		Task("format", "format code", func(_ context.Context) error { return nil }),
		Task("lint", "lint code", func(_ context.Context) error { return nil }),
		Task("test", "run tests", func(_ context.Context) error { return nil }),
	)

	data, err := ExportJSON(root)
	if err != nil {
		t.Fatalf("ExportJSON() failed: %v", err)
	}

	var tasks []TaskInfo
	if err := json.Unmarshal(data, &tasks); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	// Verify JSON structure
	for _, task := range tasks {
		if task.Name == "" {
			t.Error("task name should not be empty")
		}
		if task.Usage == "" {
			t.Error("task usage should not be empty")
		}
		if len(task.Paths) == 0 {
			t.Error("task paths should not be empty")
		}
	}
}
