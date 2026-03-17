package pk

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/fredrikaverpil/pocket/pk/internal/ctxkey"
	pkrun "github.com/fredrikaverpil/pocket/pk/run"
)

// integrationCtx creates a context suitable for integration tests with
// an execution tracker, a plan, and a buffered output.
func integrationCtx(t *testing.T, plan *Plan) (context.Context, *bytes.Buffer) {
	t.Helper()
	var stdout bytes.Buffer
	out := &pkrun.Output{Stdout: &stdout, Stderr: &stdout}

	ctx := context.Background()
	ctx = withExecutionTracker(ctx, newExecutionTracker())
	ctx = context.WithValue(ctx, ctxkey.Plan{}, plan)
	ctx = context.WithValue(ctx, ctxkey.Output{}, out)
	return ctx, &stdout
}

func TestIntegration_SerialOrder(t *testing.T) {
	var order []string
	a := &Task{Name: "a", Usage: "a", Do: func(_ context.Context) error { order = append(order, "a"); return nil }}
	b := &Task{Name: "b", Usage: "b", Do: func(_ context.Context) error { order = append(order, "b"); return nil }}
	c := &Task{Name: "c", Usage: "c", Do: func(_ context.Context) error { order = append(order, "c"); return nil }}

	cfg := &Config{Auto: Serial(a, b, c)}
	plan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	ctx, _ := integrationCtx(t, plan)
	if err := cfg.Auto.run(ctx); err != nil {
		t.Fatal(err)
	}

	if len(order) != 3 || order[0] != "a" || order[1] != "b" || order[2] != "c" {
		t.Errorf("expected [a b c], got %v", order)
	}
}

func TestIntegration_MixedComposition(t *testing.T) {
	var order []string
	var count atomic.Int32 // track parallel execution

	a := &Task{Name: "a", Usage: "first", Do: func(_ context.Context) error {
		order = append(order, "a")
		return nil
	}}
	b := &Task{Name: "b", Usage: "parallel-1", Do: func(_ context.Context) error {
		count.Add(1)
		return nil
	}}
	c := &Task{Name: "c", Usage: "parallel-2", Do: func(_ context.Context) error {
		count.Add(1)
		return nil
	}}
	d := &Task{Name: "d", Usage: "last", Do: func(_ context.Context) error {
		order = append(order, "d")
		// By this point both b and c should have completed.
		if got := count.Load(); got != 2 {
			t.Errorf("expected b and c to have run before d, count=%d", got)
		}
		return nil
	}}

	cfg := &Config{Auto: Serial(a, Parallel(b, c), d)}
	plan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	ctx, _ := integrationCtx(t, plan)
	if err := cfg.Auto.run(ctx); err != nil {
		t.Fatal(err)
	}

	// a must be first, d must be last.
	if len(order) != 2 || order[0] != "a" || order[1] != "d" {
		t.Errorf("expected order [a d], got %v", order)
	}
}

func TestIntegration_PathFilterWithDetect(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directories with marker files.
	for _, d := range []string{"svc-a", "svc-b", "lib-c"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Only svc-a and svc-b have a marker.
	if err := os.WriteFile(filepath.Join(tmpDir, "svc-a", "go.mod"), []byte("module svc-a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "svc-b", "go.mod"), []byte("module svc-b"), 0o644); err != nil {
		t.Fatal(err)
	}

	var ranPaths []string
	task := &Task{Name: "test", Usage: "test", Do: func(ctx context.Context) error {
		ranPaths = append(ranPaths, pkrun.PathFromContext(ctx))
		return nil
	}}

	cfg := &Config{
		Auto: WithOptions(task, WithDetect(DetectByFile("go.mod"))),
	}

	allDirs := []string{".", "svc-a", "svc-b", "lib-c"}
	plan, err := newPlan(cfg, tmpDir, allDirs)
	if err != nil {
		t.Fatal(err)
	}

	ctx, _ := integrationCtx(t, plan)
	if err := cfg.Auto.run(ctx); err != nil {
		t.Fatal(err)
	}

	if len(ranPaths) != 2 {
		t.Fatalf("expected 2 paths, got %v", ranPaths)
	}
}

func TestIntegration_DeduplicationAcrossComposition(t *testing.T) {
	var count atomic.Int32
	shared := &Task{Name: "shared", Usage: "shared", Do: func(_ context.Context) error {
		count.Add(1)
		return nil
	}}

	cfg := &Config{
		Auto: Serial(shared, Parallel(shared, shared)),
	}
	plan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	ctx, _ := integrationCtx(t, plan)
	if err := cfg.Auto.run(ctx); err != nil {
		t.Fatal(err)
	}

	// Same task at same path should only run once.
	if got := count.Load(); got != 1 {
		t.Errorf("expected 1 (deduplicated), got %d", got)
	}
}

type pyTestIntFlags struct {
	Python string `flag:"python" usage:"python version"`
}

func TestIntegration_WithNameSuffix_MultiVersion(t *testing.T) {
	var captured39, captured310 string

	task := &Task{
		Name:  "py-test",
		Usage: "python test",
		Flags: pyTestIntFlags{Python: "unset"},
		Do: func(ctx context.Context) error {
			ver := pkrun.GetFlags[pyTestIntFlags](ctx).Python
			suffix := nameSuffixFromContext(ctx)
			switch suffix {
			case "3.9":
				captured39 = ver
			case "3.10":
				captured310 = ver
			}
			return nil
		},
	}

	cfg := &Config{
		Auto: Serial(
			WithOptions(task, WithNameSuffix("3.9"), WithFlags(pyTestIntFlags{Python: "3.9"})),
			WithOptions(task, WithNameSuffix("3.10"), WithFlags(pyTestIntFlags{Python: "3.10"})),
		),
	}

	plan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	ctx, _ := integrationCtx(t, plan)
	if err := cfg.Auto.run(ctx); err != nil {
		t.Fatal(err)
	}

	if captured39 != "3.9" {
		t.Errorf("expected 3.9, got %q", captured39)
	}
	if captured310 != "3.10" {
		t.Errorf("expected 3.10, got %q", captured310)
	}
}

func TestIntegration_WithSkipTaskPattern(t *testing.T) {
	var lintPaths, testPaths []string

	lint := &Task{Name: "lint", Usage: "lint", Do: func(ctx context.Context) error {
		lintPaths = append(lintPaths, pkrun.PathFromContext(ctx))
		return nil
	}}
	test := &Task{Name: "test", Usage: "test", Do: func(ctx context.Context) error {
		testPaths = append(testPaths, pkrun.PathFromContext(ctx))
		return nil
	}}

	cfg := &Config{
		Auto: WithOptions(
			Parallel(lint, test),
			WithPath("svc-a", "svc-b"),
			WithSkipTask(test, "svc-b"), // test excluded from svc-b
		),
	}

	allDirs := []string{".", "svc-a", "svc-b"}
	plan, err := newPlan(cfg, "/tmp", allDirs)
	if err != nil {
		t.Fatal(err)
	}

	ctx, _ := integrationCtx(t, plan)
	if err := cfg.Auto.run(ctx); err != nil {
		t.Fatal(err)
	}

	// lint should run in both svc-a and svc-b.
	if len(lintPaths) != 2 {
		t.Errorf("expected lint in 2 paths, got %v", lintPaths)
	}

	// test should only run in svc-a (excluded from svc-b).
	if len(testPaths) != 1 {
		t.Errorf("expected test in 1 path, got %v", testPaths)
	}
	if len(testPaths) == 1 && testPaths[0] != "svc-a" {
		t.Errorf("expected test in svc-a, got %v", testPaths)
	}
}

func TestIntegration_WithSkipTask(t *testing.T) {
	a := &Task{Name: "a", Usage: "a", Do: func(_ context.Context) error { return nil }}
	b := &Task{Name: "b", Usage: "b", Do: func(_ context.Context) error { return nil }}
	c := &Task{Name: "c", Usage: "c", Do: func(_ context.Context) error { return nil }}

	cfg := &Config{
		Auto: WithOptions(
			Serial(a, b, c),
			WithSkipTask(b),
		),
	}

	plan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	// WithSkipTask removes the task from plan introspection.
	tasks := plan.Tasks()
	for _, ti := range tasks {
		if ti.Name == "b" {
			t.Error("task 'b' should not appear in plan tasks")
		}
	}

	// Only a and c should be in the plan.
	names := make(map[string]bool)
	for _, ti := range tasks {
		names[ti.Name] = true
	}
	if !names["a"] || !names["c"] {
		t.Errorf("expected a and c in plan, got %v", names)
	}
}

func TestIntegration_ManualTaskSkippedInAutoExec(t *testing.T) {
	var autoRan, manualRan bool

	autoTask := &Task{Name: "auto-task", Usage: "auto", Do: func(_ context.Context) error {
		autoRan = true
		return nil
	}}
	manualTask := &Task{Name: "manual-task", Usage: "manual", Do: func(_ context.Context) error {
		manualRan = true
		return nil
	}}

	cfg := &Config{
		Auto:   Serial(autoTask),
		Manual: []Runnable{manualTask},
	}

	plan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	// Simulate auto execution (bare `./pok` invocation).
	ctx, _ := integrationCtx(t, plan)
	ctx = context.WithValue(ctx, ctxkey.AutoExec{}, true)

	// Run auto tree.
	if err := cfg.Auto.run(ctx); err != nil {
		t.Fatal(err)
	}

	if !autoRan {
		t.Error("auto task should have run")
	}

	// Manual task should be skipped in auto exec mode.
	if err := manualTask.run(ctx); err != nil {
		t.Fatal(err)
	}
	if manualRan {
		t.Error("manual task should be skipped in auto exec mode")
	}
}

type modeIntFlags struct {
	Mode string `flag:"mode" usage:"mode"`
}

func TestIntegration_FlagOverrideViaWithFlag(t *testing.T) {
	var captured string
	task := &Task{
		Name:  "test",
		Usage: "test",
		Flags: modeIntFlags{Mode: "default"},
		Do: func(ctx context.Context) error {
			captured = pkrun.GetFlags[modeIntFlags](ctx).Mode
			return nil
		},
	}

	cfg := &Config{
		Auto: WithOptions(task, WithFlags(modeIntFlags{Mode: "overridden"})),
	}

	plan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	ctx, _ := integrationCtx(t, plan)
	if err := cfg.Auto.run(ctx); err != nil {
		t.Fatal(err)
	}

	if captured != "overridden" {
		t.Errorf("expected %q, got %q", "overridden", captured)
	}
}
