package pocket

import (
	"context"
	"testing"
)

func TestRun_ExecutesCommand(t *testing.T) {
	// Run is hard to test without executing real commands,
	// so we test that it skips in collect mode.

	cmd := Run("echo", "hello")

	// Should skip in collect mode
	out := StdOutput()
	plan := newExecutionPlan()
	ec := &execContext{
		mode:  modeCollect,
		out:   out,
		cwd:   ".",
		dedup: newDedupState(),
		plan:  plan,
	}
	ctx := withExecContext(context.Background(), ec)

	if err := cmd.run(ctx); err != nil {
		t.Errorf("Run.run() in collect mode = %v, want nil", err)
	}
}

func TestDo_ExecutesFunction(t *testing.T) {
	executed := false

	fn := Do(func(_ context.Context) error {
		executed = true
		return nil
	})

	// Should skip in collect mode
	out := StdOutput()
	plan := newExecutionPlan()
	ec := &execContext{
		mode:  modeCollect,
		out:   out,
		cwd:   ".",
		dedup: newDedupState(),
		plan:  plan,
	}
	ctx := withExecContext(context.Background(), ec)

	if err := fn.run(ctx); err != nil {
		t.Errorf("Do.run() in collect mode = %v, want nil", err)
	}

	if executed {
		t.Error("Do function was executed in collect mode, should be skipped")
	}

	// Should execute in execute mode
	ec = newExecContext(out, ".", false)
	ctx = withExecContext(context.Background(), ec)

	if err := fn.run(ctx); err != nil {
		t.Errorf("Do.run() in execute mode = %v, want nil", err)
	}

	if !executed {
		t.Error("Do function was not executed in execute mode")
	}
}

func TestDo_ComposesWithSerial(t *testing.T) {
	var order []string

	step1 := Do(func(_ context.Context) error {
		order = append(order, "step1")
		return nil
	})

	step2 := Do(func(_ context.Context) error {
		order = append(order, "step2")
		return nil
	})

	composed := Serial(step1, step2)

	out := StdOutput()
	ec := newExecContext(out, ".", false)
	ctx := withExecContext(context.Background(), ec)

	if err := composed.run(ctx); err != nil {
		t.Fatalf("Serial(Do, Do).run() = %v, want nil", err)
	}

	if len(order) != 2 {
		t.Errorf("expected 2 executions, got %d", len(order))
	}
	if order[0] != "step1" || order[1] != "step2" {
		t.Errorf("wrong execution order: %v", order)
	}
}

func TestRun_ComposesWithFuncDef(t *testing.T) {
	// Test that Run can be used as the body of a TaskDef
	// This will fail to execute (no such command) but tests composition

	taskFunc := Task("test-task", "run test command", Run("echo", "hello"))

	// Should have the TaskDef in plan.TaskDefs()
	engine := NewEngine(taskFunc)
	plan, err := engine.Plan(context.Background())
	if err != nil {
		t.Fatalf("Engine.Plan() failed: %v", err)
	}
	funcs := plan.TaskDefs()
	if len(funcs) != 1 {
		t.Errorf("expected 1 func, got %d", len(funcs))
	}
	if funcs[0].Name() != "test-task" {
		t.Errorf("expected name 'test-task', got %q", funcs[0].Name())
	}
}
