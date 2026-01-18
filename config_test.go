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

func TestSerial_TaskDefs(t *testing.T) {
	t.Parallel()

	fn1 := Task("test-format", "format test files", func(_ context.Context) error { return nil })
	fn2 := Task("test-lint", "lint test files", func(_ context.Context) error { return nil })

	runnable := Serial(fn1, fn2)
	// Check Engine.Plan().TaskDefs() returns both funcs.
	engine := NewEngine(runnable)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		t.Fatalf("Engine.Plan() failed: %v", err)
	}
	funcs := plan.TaskDefs()
	if len(funcs) != 2 {
		t.Errorf("TaskDefs() length = %d, want 2", len(funcs))
	}
}

func TestParallel_TaskDefs(t *testing.T) {
	t.Parallel()

	fn1 := Task("fn1", "func 1", func(_ context.Context) error { return nil })
	fn2 := Task("fn2", "func 2", func(_ context.Context) error { return nil })

	runnable := Parallel(fn1, fn2)
	engine := NewEngine(runnable)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		t.Fatalf("Engine.Plan() failed: %v", err)
	}
	funcs := plan.TaskDefs()
	if len(funcs) != 2 {
		t.Errorf("TaskDefs() length = %d, want 2", len(funcs))
	}
}

func TestConfig_AutoRun(t *testing.T) {
	t.Parallel()

	fn1 := Task("deploy", "deploy app", func(_ context.Context) error { return nil })
	fn2 := Task("release", "release app", func(_ context.Context) error { return nil })

	cfg := Config{
		AutoRun: Serial(fn1, fn2),
	}

	engine := NewEngine(cfg.AutoRun)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		t.Fatalf("Engine.Plan() failed: %v", err)
	}
	funcs := plan.TaskDefs()
	if len(funcs) != 2 {
		t.Errorf("AutoRun TaskDefs() length = %d, want 2", len(funcs))
	}
}

func TestNested_Serial_Parallel(t *testing.T) {
	t.Parallel()

	fn1 := Task("fn1", "func 1", func(_ context.Context) error { return nil })
	fn2 := Task("fn2", "func 2", func(_ context.Context) error { return nil })
	fn3 := Task("fn3", "func 3", func(_ context.Context) error { return nil })

	runnable := Serial(
		fn1,
		Parallel(fn2, fn3),
	)
	engine := NewEngine(runnable)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		t.Fatalf("Engine.Plan() failed: %v", err)
	}
	funcs := plan.TaskDefs()
	if len(funcs) != 3 {
		t.Errorf("TaskDefs() length = %d, want 3", len(funcs))
	}
}

// TestFuncDef_Clone_Named verifies that Clone with Named creates a copy with a new name.
func TestFuncDef_Clone_Named(t *testing.T) {
	t.Parallel()

	original := Task("go-test", "run tests", func(_ context.Context) error { return nil })
	renamed := Clone(original, Named("integration-test"))

	// Verify names are different
	if original.Name() != "go-test" {
		t.Errorf("original name changed: got %q, want %q", original.Name(), "go-test")
	}
	if renamed.Name() != "integration-test" {
		t.Errorf("renamed name wrong: got %q, want %q", renamed.Name(), "integration-test")
	}

	// Verify they're different pointers (copy, not mutation)
	if original == renamed {
		t.Error("WithName should return a copy, not the same pointer")
	}

	// Verify usage is preserved
	if renamed.Usage() != original.Usage() {
		t.Errorf("usage not preserved: got %q, want %q", renamed.Usage(), original.Usage())
	}
}

// TestFuncDef_Clone_Usage verifies that Clone with Usage creates a copy with new help text.
func TestFuncDef_Clone_Usage(t *testing.T) {
	t.Parallel()

	original := Task("go-test", "run tests", func(_ context.Context) error { return nil })
	modified := Clone(original, Usage("run unit tests only"))

	// Verify usages are different
	if original.Usage() != "run tests" {
		t.Errorf("original usage changed: got %q", original.Usage())
	}
	if modified.Usage() != "run unit tests only" {
		t.Errorf("modified usage wrong: got %q", modified.Usage())
	}

	// Verify name is preserved
	if modified.Name() != original.Name() {
		t.Errorf("name not preserved: got %q, want %q", modified.Name(), original.Name())
	}
}

// TestFuncDef_Clone_Multiple verifies that Clone accepts multiple options.
func TestFuncDef_Clone_Multiple(t *testing.T) {
	t.Parallel()

	original := Task("go-test", "run tests", func(_ context.Context) error { return nil })
	chained := Clone(original, Named("integration-test"), Usage("run integration tests"), AsHidden())

	if chained.Name() != "integration-test" {
		t.Errorf("name wrong: got %q", chained.Name())
	}
	if chained.Usage() != "run integration tests" {
		t.Errorf("usage wrong: got %q", chained.Usage())
	}
	if !chained.IsHidden() {
		t.Error("expected hidden to be true")
	}

	// Original unchanged
	if original.Name() != "go-test" || original.Usage() != "run tests" || original.IsHidden() {
		t.Error("original was mutated")
	}
}

// TestSkipEverywhere verifies that Skip(task) with no paths excludes the task from TaskDefs.
func TestSkipEverywhere(t *testing.T) {
	t.Parallel()

	task1 := Task("task1", "task 1", func(_ context.Context) error { return nil })
	task2 := Task("task2", "task 2", func(_ context.Context) error { return nil })
	workflow := Serial(task1, task2)

	// Skip task2 everywhere (no paths specified)
	runnable := RunIn(workflow, Include("."), Skip(task2))

	engine := NewEngine(runnable)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		t.Fatalf("Engine.Plan() failed: %v", err)
	}

	funcs := plan.TaskDefs()

	// Should only have task1, not task2
	if len(funcs) != 1 {
		t.Errorf("expected 1 func, got %d", len(funcs))
	}
	if len(funcs) > 0 && funcs[0].name != "task1" {
		t.Errorf("expected task1, got %s", funcs[0].name)
	}

	// Verify task2 is not in the list
	for _, f := range funcs {
		if f.name == "task2" {
			t.Error("task2 should be excluded (skipped everywhere)")
		}
	}
}

// TestSkipTaskWithManualRun_WithName verifies the documented pattern of using
// Skip + ManualRun with WithName to avoid duplicate function names.
//
// Pattern:
//
//	AutoRun: pocket.RunIn(golang.Tasks(), pocket.Include("services/api"), pocket.Skip(golang.Test, "services/api")),
//	ManualRun: []pocket.Runnable{pocket.RunIn(golang.Test.WithName("integration-test"), pocket.Include("services/api"))},
func TestSkipTaskWithManualRun_WithName(t *testing.T) {
	t.Parallel()

	// Simulate golang.Test and golang.Workflow
	testTask := Task("go-test", "test", func(_ context.Context) error { return nil })
	formatTask := Task("go-format", "format", func(_ context.Context) error { return nil })
	workflow := Serial(formatTask, testTask)

	// Use Clone with Named to give the ManualRun task a distinct name
	cfg := Config{
		AutoRun: RunIn(workflow, Include("services/api"), Skip(testTask, "services/api")),
		ManualRun: []Runnable{
			RunIn(Clone(testTask, Named("integration-test")), Include("services/api")),
		},
	}

	// Collect TaskDefs as runner.go does via Engine.Plan().TaskDefs()
	var allFuncs []*TaskDef
	if cfg.AutoRun != nil {
		engine := NewEngine(cfg.AutoRun)
		plan, err := engine.Plan(context.Background())
		if err != nil {
			t.Fatalf("Engine.Plan() for AutoRun failed: %v", err)
		}
		allFuncs = append(allFuncs, plan.TaskDefs()...)
	}
	for _, r := range cfg.ManualRun {
		engine := NewEngine(r)
		plan, err := engine.Plan(context.Background())
		if err != nil {
			t.Fatalf("Engine.Plan() for ManualRun failed: %v", err)
		}
		allFuncs = append(allFuncs, plan.TaskDefs()...)
	}

	// Count occurrences of each function name
	counts := make(map[string]int)
	for _, f := range allFuncs {
		counts[f.name]++
	}

	t.Logf("Function counts: %v", counts)

	// Check for duplicates
	for name, count := range counts {
		if count > 1 {
			t.Errorf("function %q appears %d times", name, count)
		}
	}

	// Verify all three funcs are present with unique names
	if counts["go-test"] != 1 {
		t.Errorf("expected go-test once, got %d", counts["go-test"])
	}
	if counts["go-format"] != 1 {
		t.Errorf("expected go-format once, got %d", counts["go-format"])
	}
	if counts["integration-test"] != 1 {
		t.Errorf("expected integration-test once, got %d", counts["integration-test"])
	}
}
