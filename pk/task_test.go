package pk

import (
	"context"
	"sync/atomic"
	"testing"
)

func TestTask_Run_Deduplication(t *testing.T) {
	var runCount atomic.Int32

	task := &Task{Name: "dedup-task", Usage: "test task", Do: func(_ context.Context) error {
		runCount.Add(1)
		return nil
	}}

	// Create context with tracker.
	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// First run should execute.
	if err := task.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after first run, got %d", got)
	}

	// Second run should be skipped (same task+path, deduplicated).
	if err := task.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after second run (should skip), got %d", got)
	}

	// Run with different path context should execute (different path).
	ctxServices := ContextWithPath(ctx, "services")
	if err := task.run(ctxServices); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after run with different path, got %d", got)
	}

	// Run again with services path should be skipped (same task+path).
	if err := task.run(ctxServices); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after second run with services path (should skip), got %d", got)
	}
}

func TestTask_Run_ForceRun(t *testing.T) {
	var runCount atomic.Int32

	task := &Task{Name: "force-task", Usage: "test task", Do: func(_ context.Context) error {
		runCount.Add(1)
		return nil
	}}

	// Create context with tracker.
	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// First run should execute.
	if err := task.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after first run, got %d", got)
	}

	// Second run should be skipped.
	if err := task.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after second run (should skip), got %d", got)
	}

	// Run with forceRun should execute.
	ctxForce := withForceRun(ctx)
	if err := task.run(ctxForce); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after run with forceRun, got %d", got)
	}
}

func TestTask_Run_NoTracker(t *testing.T) {
	var runCount atomic.Int32

	task := &Task{Name: "no-tracker-task", Usage: "test task", Do: func(_ context.Context) error {
		runCount.Add(1)
		return nil
	}}

	// Context without tracker - should always run (no deduplication).
	ctx := context.Background()

	if err := task.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := task.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 when no tracker present, got %d", got)
	}
}

func TestTask_Run_SameNameSamePathDeduplicated(t *testing.T) {
	var runCount1, runCount2 atomic.Int32

	// Two different task instances with the same name.
	task1 := &Task{Name: "same-name", Usage: "test task 1", Do: func(_ context.Context) error {
		runCount1.Add(1)
		return nil
	}}
	task2 := &Task{Name: "same-name", Usage: "test task 2", Do: func(_ context.Context) error {
		runCount2.Add(1)
		return nil
	}}

	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// First task should run.
	if err := task1.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second task with same name+path should be skipped (deduplication by name+path).
	if err := task2.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := runCount1.Load(); got != 1 {
		t.Errorf("expected task1 runCount=1, got %d", got)
	}
	if got := runCount2.Load(); got != 0 {
		t.Errorf("expected task2 runCount=0 (deduplicated), got %d", got)
	}
}

func TestTask_Fields(t *testing.T) {
	task := &Task{Name: "my-task", Usage: "my usage", Do: func(_ context.Context) error {
		return nil
	}}

	if task.Name != "my-task" {
		t.Errorf("expected Name=%q, got %q", "my-task", task.Name)
	}
	if task.Usage != "my usage" {
		t.Errorf("expected Usage=%q, got %q", "my usage", task.Usage)
	}
	if task.Hidden {
		t.Errorf("expected Hidden=false, got %v", task.Hidden)
	}
}

func TestTask_Hidden(t *testing.T) {
	task := &Task{Name: "visible-task", Usage: "test task", Do: func(_ context.Context) error {
		return nil
	}}

	// Should not be hidden.
	if task.Hidden {
		t.Error("task should not be hidden")
	}

	hiddenTask := &Task{Name: "hidden-task", Usage: "test task", Hidden: true, Do: func(_ context.Context) error {
		return nil
	}}

	// Should be hidden.
	if !hiddenTask.Hidden {
		t.Error("hidden task should be hidden")
	}
}

func TestTask_Run_FlagOverrides(t *testing.T) {
	task := &Task{
		Name:  "flag-task",
		Usage: "test task",
		Flags: map[string]FlagDef{
			"myflag": {Default: "default", Usage: "usage"},
		},
		Do: func(_ context.Context) error {
			return nil
		},
	}

	// Build flagSet.
	if err := task.buildFlagSet(); err != nil {
		t.Fatal(err)
	}

	// Helper to create a Plan with pre-computed flags for the task.
	planWithFlags := func(flags map[string]any) *Plan {
		return &Plan{
			taskIndex: map[string]*taskInstance{
				"flag-task": {
					task:  task,
					name:  "flag-task",
					flags: flags,
				},
			},
		}
	}

	t.Run("DefaultValue", func(t *testing.T) {
		// Reset flag to default before test.
		if err := task.flagSet.Set("myflag", "default"); err != nil {
			t.Fatal(err)
		}

		// Capture the flag value via GetFlag from inside the task.
		var captured string
		task.Do = func(ctx context.Context) error {
			captured = GetFlag[string](ctx, "myflag")
			return nil
		}

		ctx := context.Background()
		if err := task.run(ctx); err != nil {
			t.Fatal(err)
		}
		if captured != "default" {
			t.Errorf("expected default value, got %q", captured)
		}
	})

	t.Run("WithOverride", func(t *testing.T) {
		// Reset flag to default before test.
		if err := task.flagSet.Set("myflag", "default"); err != nil {
			t.Fatal(err)
		}

		ctx := context.Background()
		// Create Plan with pre-computed flag override.
		plan := planWithFlags(map[string]any{"myflag": "overridden"})
		ctx = context.WithValue(ctx, planKey{}, plan)

		// Capture the flag value via GetFlag from inside the task.
		var captured string
		task.Do = func(ctx context.Context) error {
			captured = GetFlag[string](ctx, "myflag")
			return nil
		}

		if err := task.run(ctx); err != nil {
			t.Fatal(err)
		}
		if captured != "overridden" {
			t.Errorf("expected overridden value, got %q", captured)
		}
	})

	t.Run("PreMergedOverrides", func(t *testing.T) {
		// Reset flag to default before test.
		if err := task.flagSet.Set("myflag", "default"); err != nil {
			t.Fatal(err)
		}

		ctx := context.Background()
		plan := planWithFlags(map[string]any{"myflag": "inner"})
		ctx = context.WithValue(ctx, planKey{}, plan)

		var captured string
		task.Do = func(ctx context.Context) error {
			captured = GetFlag[string](ctx, "myflag")
			return nil
		}

		if err := task.run(ctx); err != nil {
			t.Fatal(err)
		}
		if captured != "inner" {
			t.Errorf("expected pre-merged override value, got %q", captured)
		}
	})

}

// TestTask_Run_NameSuffixDeduplication tests that tasks with same base name
// but different suffixes (via WithNameSuffix) are NOT deduplicated.
// This is critical for multi-version testing (e.g., py-test:3.9, py-test:3.10).
func TestTask_Run_NameSuffixDeduplication(t *testing.T) {
	var runCount atomic.Int32

	task := &Task{Name: "py-test", Usage: "test task", Do: func(_ context.Context) error {
		runCount.Add(1)
		return nil
	}}

	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// Run with suffix "3.9" - should execute.
	ctx39 := contextWithNameSuffix(ctx, "3.9")
	if err := task.run(ctx39); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after py-test:3.9, got %d", got)
	}

	// Run with suffix "3.10" - should also execute (different effective name).
	ctx310 := contextWithNameSuffix(ctx, "3.10")
	if err := task.run(ctx310); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after py-test:3.10, got %d", got)
	}

	// Run again with suffix "3.9" - should be skipped (duplicate).
	if err := task.run(ctx39); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after duplicate py-test:3.9, got %d", got)
	}

	// Run with suffix "3.11" - should execute.
	ctx311 := contextWithNameSuffix(ctx, "3.11")
	if err := task.run(ctx311); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 3 {
		t.Errorf("expected runCount=3 after py-test:3.11, got %d", got)
	}
}

// TestTask_Run_GlobalDeduplicationIgnoresSuffix tests that global tasks
// deduplicate by base name only, ignoring name suffix.
// This is critical for install tasks that should only run once.
func TestTask_Run_GlobalDeduplicationIgnoresSuffix(t *testing.T) {
	var runCount atomic.Int32

	// Global task - should deduplicate by base name only.
	task := &Task{Name: "install:uv", Usage: "install uv", Hidden: true, Global: true, Do: func(_ context.Context) error {
		runCount.Add(1)
		return nil
	}}

	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// Run with suffix "3.9" - should execute.
	ctx39 := contextWithNameSuffix(ctx, "3.9")
	if err := task.run(ctx39); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after first run, got %d", got)
	}

	// Run with suffix "3.10" - should be skipped (global uses base name only).
	ctx310 := contextWithNameSuffix(ctx, "3.10")
	if err := task.run(ctx310); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after second run with different suffix (global should skip), got %d", got)
	}

	// Run with different path - should still be skipped (global ignores path too).
	ctx39Services := ContextWithPath(ctx39, "services")
	if err := task.run(ctx39Services); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after run with different path (global should skip), got %d", got)
	}
}

// TestTask_Run_NonGlobalWithSuffixAndPath tests that non-global tasks
// deduplicate by (effective name, path) tuple.
func TestTask_Run_NonGlobalWithSuffixAndPath(t *testing.T) {
	var runCount atomic.Int32

	task := &Task{Name: "test", Usage: "test task", Do: func(_ context.Context) error {
		runCount.Add(1)
		return nil
	}}

	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// test:3.9 at root - should execute.
	ctx39 := contextWithNameSuffix(ctx, "3.9")
	if err := task.run(ctx39); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1, got %d", got)
	}

	// test:3.9 at services - should execute (different path).
	ctx39Services := ContextWithPath(ctx39, "services")
	if err := task.run(ctx39Services); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2, got %d", got)
	}

	// test:3.10 at services - should execute (different suffix).
	ctx310Services := ContextWithPath(contextWithNameSuffix(ctx, "3.10"), "services")
	if err := task.run(ctx310Services); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 3 {
		t.Errorf("expected runCount=3, got %d", got)
	}

	// test:3.9 at services again - should be skipped (duplicate).
	if err := task.run(ctx39Services); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 3 {
		t.Errorf("expected runCount=3 (duplicate should skip), got %d", got)
	}
}

// TestTask_Run_EffectiveName tests that the effective name (base name + suffix)
// is used correctly for task identification.
func TestTask_Run_EffectiveName(t *testing.T) {
	t.Run("NoSuffix", func(t *testing.T) {
		task := &Task{Name: "test", Usage: "test task", Do: func(_ context.Context) error {
			return nil
		}}
		ctx := context.Background()
		// Effective name should be base name when no suffix.
		effectiveName := task.Name
		if suffix := nameSuffixFromContext(ctx); suffix != "" {
			effectiveName = task.Name + ":" + suffix
		}
		if effectiveName != "test" {
			t.Errorf("expected effectiveName=%q, got %q", "test", effectiveName)
		}
	})

	t.Run("WithSuffix", func(t *testing.T) {
		task := &Task{Name: "py-test", Usage: "test task", Do: func(_ context.Context) error {
			return nil
		}}
		ctx := contextWithNameSuffix(context.Background(), "3.9")
		effectiveName := task.Name
		if suffix := nameSuffixFromContext(ctx); suffix != "" {
			effectiveName = task.Name + ":" + suffix
		}
		if effectiveName != "py-test:3.9" {
			t.Errorf("expected effectiveName=%q, got %q", "py-test:3.9", effectiveName)
		}
	})

	t.Run("NestedSuffix", func(t *testing.T) {
		task := &Task{Name: "test", Usage: "test task", Do: func(_ context.Context) error {
			return nil
		}}
		ctx := contextWithNameSuffix(context.Background(), "a")
		ctx = contextWithNameSuffix(ctx, "b")
		effectiveName := task.Name
		if suffix := nameSuffixFromContext(ctx); suffix != "" {
			effectiveName = task.Name + ":" + suffix
		}
		if effectiveName != "test:a:b" {
			t.Errorf("expected effectiveName=%q, got %q", "test:a:b", effectiveName)
		}
	})
}

func TestGetFlag(t *testing.T) {
	t.Run("FromContext", func(t *testing.T) {
		ctx := withTaskFlags(context.Background(), map[string]any{
			"name":  "hello",
			"fix":   true,
			"count": 42,
		})

		if got := GetFlag[string](ctx, "name"); got != "hello" {
			t.Errorf("expected 'hello', got %q", got)
		}
		if got := GetFlag[bool](ctx, "fix"); !got {
			t.Errorf("expected true, got %v", got)
		}
		if got := GetFlag[int](ctx, "count"); got != 42 {
			t.Errorf("expected 42, got %d", got)
		}
	})

	t.Run("MissingFlagPanics", func(t *testing.T) {
		ctx := withTaskFlags(context.Background(), map[string]any{})

		assertFlagPanic(t, func() {
			GetFlag[string](ctx, "missing")
		}, `flag "missing": not found`)
	})

	t.Run("NoFlagsInContextPanics", func(t *testing.T) {
		ctx := context.Background()

		assertFlagPanic(t, func() {
			GetFlag[string](ctx, "name")
		}, `flag "name": no flags in context`)
	})

	t.Run("TypeMismatchPanics", func(t *testing.T) {
		ctx := withTaskFlags(context.Background(), map[string]any{
			"name": 42, // int, not string
		})

		assertFlagPanic(t, func() {
			GetFlag[string](ctx, "name")
		}, `flag "name": expected string, got int`)
	})
}

// assertFlagPanic asserts that fn panics with a flagError containing wantMsg.
func assertFlagPanic(t *testing.T, fn func(), wantMsg string) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got none")
		}
		fe, ok := r.(flagError)
		if !ok {
			t.Fatalf("expected flagError panic, got %T: %v", r, r)
		}
		if got := fe.err.Error(); got != wantMsg {
			t.Errorf("expected error %q, got %q", wantMsg, got)
		}
	}()
	fn()
}

func TestBuildFlagSet(t *testing.T) {
	t.Run("AllTypes", func(t *testing.T) {
		task := &Task{
			Name: "test",
			Flags: map[string]FlagDef{
				"name":    {Default: "hello", Usage: "a name"},
				"verbose": {Default: false, Usage: "verbose mode"},
				"count":   {Default: 42, Usage: "a count"},
				"rate":    {Default: 1.5, Usage: "a rate"},
			},
			Do: func(_ context.Context) error { return nil },
		}

		if err := task.buildFlagSet(); err != nil {
			t.Fatal(err)
		}

		// Verify flags are registered.
		if f := task.flagSet.Lookup("name"); f == nil {
			t.Error("expected 'name' flag")
		}
		if f := task.flagSet.Lookup("verbose"); f == nil {
			t.Error("expected 'verbose' flag")
		}
		if f := task.flagSet.Lookup("count"); f == nil {
			t.Error("expected 'count' flag")
		}
		if f := task.flagSet.Lookup("rate"); f == nil {
			t.Error("expected 'rate' flag")
		}
	})

	t.Run("UnsupportedType", func(t *testing.T) {
		task := &Task{
			Name: "test",
			Flags: map[string]FlagDef{
				"bad": {Default: []string{"a", "b"}, Usage: "unsupported"},
			},
			Do: func(_ context.Context) error { return nil },
		}

		if err := task.buildFlagSet(); err == nil {
			t.Error("expected error for unsupported type")
		}
	})

	t.Run("EmptyFlags", func(t *testing.T) {
		task := &Task{
			Name: "test",
			Do:   func(_ context.Context) error { return nil },
		}

		if err := task.buildFlagSet(); err != nil {
			t.Fatal(err)
		}
		if task.flagSet == nil {
			t.Error("expected non-nil flagSet even with no flags")
		}
	})
}

func TestGetFlag_RecoveredByTaskRun(t *testing.T) {
	task := &Task{
		Name: "bad-flag",
		Flags: map[string]FlagDef{
			"real": {Default: "value", Usage: "a real flag"},
		},
		Do: func(ctx context.Context) error {
			_ = GetFlag[string](ctx, "typo") // This will panic.
			return nil
		},
	}

	if err := task.buildFlagSet(); err != nil {
		t.Fatal(err)
	}

	// task.run() should recover the panic and return an error.
	err := task.run(context.Background())
	if err == nil {
		t.Fatal("expected error from recovered GetFlag panic, got nil")
	}
	if got := err.Error(); got != `task "bad-flag": flag "typo": not found` {
		t.Errorf("unexpected error message: %q", got)
	}
}
