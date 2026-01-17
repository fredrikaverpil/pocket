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

func TestBuildExportPlan_Simple(t *testing.T) {
	fn := Task("build", "build project", func(_ context.Context) error {
		return nil
	})

	cfg := Config{AutoRun: fn}
	export, err := BuildExportPlan(cfg)
	if err != nil {
		t.Fatalf("BuildExportPlan() failed: %v", err)
	}

	if len(export.AutoRun) != 1 {
		t.Fatalf("expected 1 step in AutoRun, got %d", len(export.AutoRun))
	}
	if export.AutoRun[0].Name != "build" {
		t.Errorf("expected name 'build', got %q", export.AutoRun[0].Name)
	}
	if export.AutoRun[0].Type != "func" {
		t.Errorf("expected type 'func', got %q", export.AutoRun[0].Type)
	}

	// Verify it marshals to valid JSON
	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal to JSON: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON output")
	}
}

func TestBuildExportPlan_Empty(t *testing.T) {
	cfg := Config{}
	export, err := BuildExportPlan(cfg)
	if err != nil {
		t.Fatalf("BuildExportPlan() failed: %v", err)
	}

	if len(export.AutoRun) != 0 {
		t.Errorf("expected 0 AutoRun steps, got %d", len(export.AutoRun))
	}
	if len(export.ManualRun) != 0 {
		t.Errorf("expected 0 ManualRun tasks, got %d", len(export.ManualRun))
	}
}

func TestBuildExportPlan_TreeStructure(t *testing.T) {
	root := Serial(
		Task("format", "format code", func(_ context.Context) error { return nil }),
		Parallel(
			Task("lint", "lint code", func(_ context.Context) error { return nil }),
			Task("test", "run tests", func(_ context.Context) error { return nil }),
		),
	)

	cfg := Config{AutoRun: root}
	export, err := BuildExportPlan(cfg)
	if err != nil {
		t.Fatalf("BuildExportPlan() failed: %v", err)
	}

	// Should have serial at root with format and parallel children
	if len(export.AutoRun) != 1 {
		t.Fatalf("expected 1 root step, got %d", len(export.AutoRun))
	}

	serial := export.AutoRun[0]
	if serial.Type != "serial" {
		t.Errorf("expected root type 'serial', got %q", serial.Type)
	}
	if len(serial.Children) != 2 {
		t.Fatalf("expected 2 children in serial, got %d", len(serial.Children))
	}

	// First child: format func
	if serial.Children[0].Type != "func" || serial.Children[0].Name != "format" {
		t.Errorf("expected first child to be func 'format', got %s %s",
			serial.Children[0].Type, serial.Children[0].Name)
	}

	// Second child: parallel with lint and test
	parallel := serial.Children[1]
	if parallel.Type != "parallel" {
		t.Errorf("expected second child type 'parallel', got %q", parallel.Type)
	}
	if len(parallel.Children) != 2 {
		t.Fatalf("expected 2 children in parallel, got %d", len(parallel.Children))
	}
}

func TestBuildExportPlan_ManualRun(t *testing.T) {
	autoRun := Task("lint", "lint code", func(_ context.Context) error { return nil })
	manualTask := Task("deploy", "deploy app", func(_ context.Context) error { return nil })

	cfg := Config{
		AutoRun:   autoRun,
		ManualRun: []Runnable{manualTask},
	}
	export, err := BuildExportPlan(cfg)
	if err != nil {
		t.Fatalf("BuildExportPlan() failed: %v", err)
	}

	// AutoRun should have lint
	if len(export.AutoRun) != 1 {
		t.Fatalf("expected 1 AutoRun step, got %d", len(export.AutoRun))
	}
	if export.AutoRun[0].Name != "lint" {
		t.Errorf("expected AutoRun name 'lint', got %q", export.AutoRun[0].Name)
	}

	// ManualRun should have deploy as flat list
	if len(export.ManualRun) != 1 {
		t.Fatalf("expected 1 ManualRun task, got %d", len(export.ManualRun))
	}
	if export.ManualRun[0].Name != "deploy" {
		t.Errorf("expected ManualRun name 'deploy', got %q", export.ManualRun[0].Name)
	}
}
