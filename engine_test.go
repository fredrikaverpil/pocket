package pocket

import (
	"context"
	"testing"
)

func TestEngine_Plan_Simple(t *testing.T) {
	// Create a simple function
	fn := Task("test", "test function", func(_ context.Context) error {
		return nil
	})

	// Create engine and collect plan
	engine := NewEngine(fn)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan() failed: %v", err)
	}

	// Verify plan has one step
	steps := plan.Steps()
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}

	// Verify step is a function
	if steps[0].Type != "func" {
		t.Errorf("expected type 'func', got %q", steps[0].Type)
	}
	if steps[0].Name != "test" {
		t.Errorf("expected name 'test', got %q", steps[0].Name)
	}
}

func TestEngine_Plan_Serial(t *testing.T) {
	fn1 := Task("fn1", "first", func(_ context.Context) error { return nil })
	fn2 := Task("fn2", "second", func(_ context.Context) error { return nil })

	// Create serial composition
	root := Serial(fn1, fn2)

	engine := NewEngine(root)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan() failed: %v", err)
	}

	steps := plan.Steps()
	if len(steps) != 1 {
		t.Fatalf("expected 1 top-level step, got %d", len(steps))
	}

	serial := steps[0]
	if serial.Type != "serial" {
		t.Errorf("expected type 'serial', got %q", serial.Type)
	}
	if len(serial.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(serial.Children))
	}
}

func TestEngine_Plan_NestedDeps(t *testing.T) {
	// Create a hidden install function
	install := Task("install:tool", "install", func(_ context.Context) error {
		return nil
	}, AsHidden())

	// Create a function that depends on install using static composition
	// (inline Serial() calls in function bodies are not visible in plan)
	lint := Task("lint", "lint code", Serial(
		install,
		func(_ context.Context) error {
			return nil
		},
	))

	engine := NewEngine(lint)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan() failed: %v", err)
	}

	steps := plan.Steps()
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}

	// The lint function should have children (the nested Serial)
	lintStep := steps[0]
	if lintStep.Type != "func" {
		t.Errorf("expected type 'func', got %q", lintStep.Type)
	}
	if lintStep.Name != "lint" {
		t.Errorf("expected name 'lint', got %q", lintStep.Name)
	}

	// Should have a child (the Serial containing install)
	if len(lintStep.Children) != 1 {
		t.Fatalf("expected 1 child (nested Serial), got %d", len(lintStep.Children))
	}

	serialStep := lintStep.Children[0]
	if serialStep.Type != "serial" {
		t.Errorf("expected type 'serial', got %q", serialStep.Type)
	}

	// Serial should contain the install function (and anonymous function wrapper)
	if len(serialStep.Children) != 1 {
		t.Fatalf("expected 1 child in Serial (install), got %d", len(serialStep.Children))
	}

	installStep := serialStep.Children[0]
	if installStep.Name != "install:tool" {
		t.Errorf("expected name 'install:tool', got %q", installStep.Name)
	}
	if !installStep.Hidden {
		t.Error("expected install to be hidden")
	}
}

func TestEngine_Plan_Deduplication(t *testing.T) {
	// Create a shared dependency
	install := Task("install:shared", "install", func(_ context.Context) error {
		return nil
	}, AsHidden())

	// Create two functions that both depend on install using static composition
	fn1 := Task("fn1", "first", Serial(
		install,
		func(_ context.Context) error { return nil },
	))
	fn2 := Task("fn2", "second", Serial(
		install,
		func(_ context.Context) error { return nil },
	))

	root := Serial(fn1, fn2)

	engine := NewEngine(root)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan() failed: %v", err)
	}

	// Find all install steps and check dedup status
	var installSteps []*PlanStep
	var findInstalls func(steps []*PlanStep)
	findInstalls = func(steps []*PlanStep) {
		for _, s := range steps {
			if s.Name == "install:shared" {
				installSteps = append(installSteps, s)
			}
			findInstalls(s.Children)
		}
	}
	findInstalls(plan.Steps())

	if len(installSteps) != 2 {
		t.Fatalf("expected 2 install steps, got %d", len(installSteps))
	}

	// First occurrence should not be deduped
	if installSteps[0].Deduped {
		t.Error("first install occurrence should not be deduped")
	}

	// Second occurrence should be deduped
	if !installSteps[1].Deduped {
		t.Error("second install occurrence should be deduped")
	}
}

func TestEngine_Plan_NilRoot(t *testing.T) {
	engine := NewEngine(nil)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan() failed: %v", err)
	}

	steps := plan.Steps()
	if len(steps) != 0 {
		t.Errorf("expected 0 steps for nil root, got %d", len(steps))
	}
}
