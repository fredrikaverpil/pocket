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

	// Create a TaskDef using Serial for composition
	testFunc := Task("test", "test", Serial(fn1, fn2))

	// Create execution context and run
	out := StdOutput()
	ec := newExecContext(out, ".", false, nil)
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

	// Create a TaskDef using Parallel for composition
	testFunc := Task("test", "test", Parallel(fn1, fn2))

	// Create execution context and run
	out := StdOutput()
	ec := newExecContext(out, ".", false, nil)
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

func TestOptions_ShadowingPanics(t *testing.T) {
	type SharedOptions struct {
		Value string
	}

	inner := func(_ context.Context) error {
		return nil
	}
	outer := func(_ context.Context) error {
		return nil
	}

	// Create nested FuncDefs that both use the same options type
	innerFunc := Task("inner", "inner func", inner, Opts(SharedOptions{Value: "inner"}))
	outerFunc := Task("outer", "outer func", Serial(innerFunc, outer), Opts(SharedOptions{Value: "outer"}))

	// Create execution context and run - should panic
	out := StdOutput()
	ec := newExecContext(out, ".", false, nil)
	ctx := withExecContext(context.Background(), ec)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when nested functions share the same options type")
		}
	}()

	_ = outerFunc.run(ctx)
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

	// Pattern: TaskDef with install dependency
	installFunc := Task("install", "install tool", install, AsHidden())
	lintFunc := Task("lint", "run linter", Serial(installFunc, lint))

	// Create execution context and run
	out := StdOutput()
	ec := newExecContext(out, ".", false, nil)
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

// TestParallel_ClonedTasksWithDifferentOptions tests that cloned tasks with
// different options all execute their inner runnables with the correct options.
// This replicates the TestMatrix pattern used in creosote.
func TestParallel_ClonedTasksWithDifferentOptions(t *testing.T) {
	type TestOpts struct {
		Version string `arg:"version"`
	}

	// Track which versions executed in the action
	executed := make(chan string, 5)

	// Hidden install task (deduplicated - should run once)
	installRuns := make(chan struct{}, 5)
	install := Task("install", "install", Do(func(_ context.Context) error {
		installRuns <- struct{}{}
		return nil
	}), AsHidden())

	// Action reads options from context (should run for each clone)
	action := Do(func(ctx context.Context) error {
		opts := Options[TestOpts](ctx)
		executed <- opts.Version
		return nil
	})

	// Base task with Serial(install, action)
	baseTask := Task("test", "test", Serial(install, action), Opts(TestOpts{}))

	// Clone task 5 times with different versions (like TestMatrix)
	versions := []string{"3.9", "3.10", "3.11", "3.12", "3.13"}
	tasks := make([]any, len(versions))
	for i, v := range versions {
		tasks[i] = Clone(baseTask, Named("test:"+v), Opts(TestOpts{Version: v}))
	}

	// Run all clones in parallel
	parallelTasks := Parallel(tasks...)

	// Create execution context and run
	out := StdOutput()
	ec := newExecContext(out, ".", false, nil)
	ctx := withExecContext(context.Background(), ec)

	if err := parallelTasks.run(ctx); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// Check install ran only once (deduplicated)
	close(installRuns)
	installCount := 0
	for range installRuns {
		installCount++
	}
	if installCount != 1 {
		t.Errorf("expected install to run once, ran %d times", installCount)
	}

	// Check all 5 versions executed
	close(executed)
	gotVersions := make(map[string]bool)
	for v := range executed {
		gotVersions[v] = true
	}

	if len(gotVersions) != 5 {
		t.Errorf("expected 5 versions, got %d: %v", len(gotVersions), gotVersions)
	}
	for _, v := range versions {
		if !gotVersions[v] {
			t.Errorf("version %s did not execute", v)
		}
	}
}
