package pocket

import (
	"context"
	"testing"
)

func TestSerial_Composition(t *testing.T) {
	var executed []string

	fn1 := func(_ context.Context) error {
		executed = append(executed, "fn1")
		return nil
	}
	fn2 := func(_ context.Context) error {
		executed = append(executed, "fn2")
		return nil
	}

	// Create a FuncDef using Serial for composition
	testFunc := Func("test", "test", Serial(fn1, fn2))

	// Create execution context and run
	out := StdOutput()
	ec := newExecContext(out, ".", false)
	ctx := withExecContext(context.Background(), ec)

	if err := testFunc.run(ctx); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if len(executed) != 2 {
		t.Errorf("expected 2 executions, got %d", len(executed))
	}
	if executed[0] != "fn1" || executed[1] != "fn2" {
		t.Errorf("wrong execution order: %v", executed)
	}
}

func TestParallel_Composition(t *testing.T) {
	executed := make(chan string, 2)

	fn1 := func(_ context.Context) error {
		executed <- "fn1"
		return nil
	}
	fn2 := func(_ context.Context) error {
		executed <- "fn2"
		return nil
	}

	// Create a FuncDef using Parallel for composition
	testFunc := Func("test", "test", Parallel(fn1, fn2))

	// Create execution context and run
	out := StdOutput()
	ec := newExecContext(out, ".", false)
	ctx := withExecContext(context.Background(), ec)

	if err := testFunc.run(ctx); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	close(executed)
	results := make([]string, 0, 2)
	for s := range executed {
		results = append(results, s)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 executions, got %d", len(results))
	}
}

func TestSerial_WithDependency(t *testing.T) {
	var executed []string

	install := func(_ context.Context) error {
		executed = append(executed, "install")
		return nil
	}
	lint := func(_ context.Context) error {
		executed = append(executed, "lint")
		return nil
	}

	// Pattern: FuncDef with install dependency
	installFunc := Func("install", "install tool", install).Hidden()
	lintFunc := Func("lint", "run linter", Serial(installFunc, lint))

	// Create execution context and run
	out := StdOutput()
	ec := newExecContext(out, ".", false)
	ctx := withExecContext(context.Background(), ec)

	if err := lintFunc.run(ctx); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if len(executed) != 2 {
		t.Errorf("expected 2 executions, got %d", len(executed))
	}
	if executed[0] != "install" || executed[1] != "lint" {
		t.Errorf("wrong execution order: %v", executed)
	}
}
