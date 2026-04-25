package pk

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/fredrikaverpil/pocket/pk/internal/ctxkey"
	"github.com/fredrikaverpil/pocket/pk/repopath"
	pkrun "github.com/fredrikaverpil/pocket/pk/run"
	"gotest.tools/v3/assert"
)

// Flag struct types for test tasks.
type flaggedFlags struct {
	Mode  string `flag:"mode"  usage:"mode flag"`
	Count int    `flag:"count" usage:"count flag"`
}

type cliFlaggedFlags struct {
	Mode string `flag:"mode" usage:"mode flag"`
}

type versionedFlags struct {
	Ver string `flag:"ver" usage:"version"`
}

// execRecord captures the context a task received at execution time.
type execRecord struct {
	TaskName string
	Path     string
	Flags    map[string]any
	Env      pkrun.EnvConfig
}

// recorder collects execution records from spy tasks.
type recorder struct {
	mu      sync.Mutex
	records []execRecord
}

func newRecorder() *recorder {
	return &recorder{}
}

// task creates a spy task that records its execution context.
func (r *recorder) task(name string) *Task {
	return &Task{
		Name:  name,
		Usage: name,
		Do: func(ctx context.Context) error {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.records = append(r.records, execRecord{
				TaskName: name,
				Path:     pkrun.PathFromContext(ctx),
				Env:      pkrun.EnvConfigFromContext(ctx),
			})
			return nil
		},
	}
}

// failTask creates a spy task that records, then returns an error.
func (r *recorder) failTask(name string) *Task {
	return &Task{
		Name:  name,
		Usage: name,
		Do: func(ctx context.Context) error {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.records = append(r.records, execRecord{
				TaskName: name,
				Path:     pkrun.PathFromContext(ctx),
			})
			return fmt.Errorf("task %s failed", name)
		},
	}
}

// taskWithFlags creates a spy task with flags that records flag values.
func (r *recorder) taskWithFlags(name string, flags any) *Task {
	return &Task{
		Name:  name,
		Usage: name,
		Flags: flags,
		Do: func(ctx context.Context) error {
			resolved := ctx.Value(ctxkey.TaskFlags{}).(map[string]any)
			captured := make(map[string]any, len(resolved))
			maps.Copy(captured, resolved)
			r.mu.Lock()
			defer r.mu.Unlock()
			r.records = append(r.records, execRecord{
				TaskName: name,
				Path:     pkrun.PathFromContext(ctx),
				Flags:    captured,
				Env:      pkrun.EnvConfigFromContext(ctx),
			})
			return nil
		},
	}
}

// e2eSetup creates a temp dir and overrides the git root for the test.
func e2eSetup(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	repopath.SetGitRootFunc(func() string { return tmpDir })
	t.Cleanup(func() { repopath.SetGitRootFunc(nil) })
	return tmpDir
}

// e2eCtx creates a context with tracker, plan, and buffered output.
// Reuses integrationCtx from integration_test.go.
func e2eCtx(t *testing.T, plan *Plan) context.Context {
	t.Helper()
	ctx, _ := integrationCtx(t, plan)
	return ctx
}

// --- ExecuteTask public API tests ---

func TestE2E_ExecuteTask_Basic(t *testing.T) {
	e2eSetup(t)
	rec := newRecorder()
	task := rec.task("basic")

	cfg := &Config{Auto: task}
	plan, err := newPublicPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := e2eCtx(t, plan)
	if err := ExecuteTask(ctx, "basic", plan); err != nil {
		t.Fatal(err)
	}

	if len(rec.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(rec.records))
	}
	if rec.records[0].TaskName != "basic" {
		t.Errorf("expected task name %q, got %q", "basic", rec.records[0].TaskName)
	}
	if rec.records[0].Path != "." {
		t.Errorf("expected path %q, got %q", ".", rec.records[0].Path)
	}
}

func TestE2E_ExecuteTask_PathScoping(t *testing.T) {
	tmpDir := e2eSetup(t)

	// Create subdirs.
	for _, d := range []string{"svc-a", "svc-b"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	rec := newRecorder()
	task := rec.task("scoped")

	cfg := &Config{Auto: WithOptions(task, WithPath("svc-a", "svc-b"))}
	plan, err := newPublicPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := e2eCtx(t, plan)
	if err := ExecuteTask(ctx, "scoped", plan); err != nil {
		t.Fatal(err)
	}

	if len(rec.records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(rec.records))
	}
	paths := map[string]bool{}
	for _, r := range rec.records {
		paths[r.Path] = true
	}
	if !paths["svc-a"] || !paths["svc-b"] {
		t.Errorf("expected paths svc-a and svc-b, got %v", paths)
	}
}

func TestE2E_ExecuteTask_FlagResolution(t *testing.T) {
	e2eSetup(t)
	rec := newRecorder()
	task := rec.taskWithFlags("flagged", flaggedFlags{Mode: "default-mode", Count: 1})

	cfg := &Config{
		Auto: WithOptions(task, WithFlags(flaggedFlags{Mode: "overridden", Count: 1})),
	}
	plan, err := newPublicPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := e2eCtx(t, plan)
	if err := ExecuteTask(ctx, "flagged", plan); err != nil {
		t.Fatal(err)
	}

	if len(rec.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(rec.records))
	}
	r := rec.records[0]
	if r.Flags["mode"] != "overridden" {
		t.Errorf("expected mode=%q, got %q", "overridden", r.Flags["mode"])
	}
	if r.Flags["count"] != 1 {
		t.Errorf("expected count=1, got %v", r.Flags["count"])
	}
}

func TestE2E_ExecuteTask_CLIFlags(t *testing.T) {
	e2eSetup(t)
	rec := newRecorder()
	task := rec.taskWithFlags("cli-flagged", cliFlaggedFlags{Mode: "default"})

	cfg := &Config{
		Auto: WithOptions(task, WithFlags(cliFlaggedFlags{Mode: "plan-override"})),
	}
	plan, err := newPublicPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := e2eCtx(t, plan)
	// CLI flags have highest priority.
	ctx = context.WithValue(ctx, ctxkey.CLIFlags{}, map[string]any{"mode": "cli-value"})

	if err := ExecuteTask(ctx, "cli-flagged", plan); err != nil {
		t.Fatal(err)
	}

	if len(rec.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(rec.records))
	}
	if rec.records[0].Flags["mode"] != "cli-value" {
		t.Errorf("expected mode=%q, got %q", "cli-value", rec.records[0].Flags["mode"])
	}
}

func TestE2E_ExecuteTask_EnvPropagation(t *testing.T) {
	e2eSetup(t)
	rec := newRecorder()
	task := rec.task("env-task")

	cfg := &Config{Auto: task}
	plan, err := newPublicPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := e2eCtx(t, plan)
	ctx = pkrun.ContextWithEnv(ctx, "MY_VAR=hello")
	ctx = pkrun.ContextWithEnv(ctx, "OTHER=world")

	if err := ExecuteTask(ctx, "env-task", plan); err != nil {
		t.Fatal(err)
	}

	if len(rec.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(rec.records))
	}
	env := rec.records[0].Env
	if env.Set["MY_VAR"] != "hello" {
		t.Errorf("expected MY_VAR=hello, got %q", env.Set["MY_VAR"])
	}
	if env.Set["OTHER"] != "world" {
		t.Errorf("expected OTHER=world, got %q", env.Set["OTHER"])
	}
}

func TestE2E_ExecuteTask_NameSuffix(t *testing.T) {
	e2eSetup(t)
	rec := newRecorder()
	task := rec.taskWithFlags("versioned", versionedFlags{Ver: "unset"})

	cfg := &Config{
		Auto: WithOptions(task, WithNameSuffix("3.9"), WithFlags(versionedFlags{Ver: "3.9"})),
	}
	plan, err := newPublicPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := e2eCtx(t, plan)
	if err := ExecuteTask(ctx, "versioned:3.9", plan); err != nil {
		t.Fatal(err)
	}

	if len(rec.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(rec.records))
	}
	if rec.records[0].Flags["ver"] != "3.9" {
		t.Errorf("expected ver=%q, got %q", "3.9", rec.records[0].Flags["ver"])
	}
}

func TestE2E_ExecuteTask_NotFound(t *testing.T) {
	e2eSetup(t)
	cfg := &Config{Auto: &Task{
		Name:  "exists",
		Usage: "exists",
		Do:    func(context.Context) error { return nil },
	}}
	plan, err := newPublicPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := e2eCtx(t, plan)
	err = ExecuteTask(ctx, "nonexistent", plan)
	if err == nil {
		t.Fatal("expected error for unknown task")
	}
}

// --- Auto execution tests ---

func TestE2E_AutoExec_SerialOrder(t *testing.T) {
	e2eSetup(t)
	rec := newRecorder()
	a, b, c := rec.task("a"), rec.task("b"), rec.task("c")

	cfg := &Config{Auto: Serial(a, b, c)}
	plan, err := newPublicPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := e2eCtx(t, plan)
	ctx = context.WithValue(ctx, ctxkey.AutoExec{}, true)
	if err := cfg.Auto.run(ctx); err != nil {
		t.Fatal(err)
	}

	if len(rec.records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(rec.records))
	}
	for i, name := range []string{"a", "b", "c"} {
		if rec.records[i].TaskName != name {
			t.Errorf("record[%d]: expected %q, got %q", i, name, rec.records[i].TaskName)
		}
	}
}

func TestE2E_AutoExec_ParallelAll(t *testing.T) {
	e2eSetup(t)
	rec := newRecorder()
	a, b, c := rec.task("a"), rec.task("b"), rec.task("c")

	cfg := &Config{Auto: Parallel(a, b, c)}
	plan, err := newPublicPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := e2eCtx(t, plan)
	ctx = context.WithValue(ctx, ctxkey.AutoExec{}, true)
	if err := cfg.Auto.run(ctx); err != nil {
		t.Fatal(err)
	}

	if len(rec.records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(rec.records))
	}
	names := map[string]bool{}
	for _, r := range rec.records {
		names[r.TaskName] = true
	}
	for _, name := range []string{"a", "b", "c"} {
		if !names[name] {
			t.Errorf("expected task %q to have run", name)
		}
	}
}

func TestE2E_AutoExec_MixedComposition(t *testing.T) {
	e2eSetup(t)
	rec := newRecorder()
	a, b, c, d := rec.task("a"), rec.task("b"), rec.task("c"), rec.task("d")

	cfg := &Config{Auto: Serial(a, Parallel(b, c), d)}
	plan, err := newPublicPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := e2eCtx(t, plan)
	ctx = context.WithValue(ctx, ctxkey.AutoExec{}, true)
	if err := cfg.Auto.run(ctx); err != nil {
		t.Fatal(err)
	}

	if len(rec.records) != 4 {
		t.Fatalf("expected 4 records, got %d", len(rec.records))
	}
	// A must be first, D must be last.
	if rec.records[0].TaskName != "a" {
		t.Errorf("expected first task %q, got %q", "a", rec.records[0].TaskName)
	}
	if rec.records[3].TaskName != "d" {
		t.Errorf("expected last task %q, got %q", "d", rec.records[3].TaskName)
	}
	// B and C must both be present in the middle.
	middle := map[string]bool{rec.records[1].TaskName: true, rec.records[2].TaskName: true}
	if !middle["b"] || !middle["c"] {
		t.Errorf("expected b and c in middle, got %v", middle)
	}
}

func TestE2E_AutoExec_SerialStopsOnError(t *testing.T) {
	e2eSetup(t)
	rec := newRecorder()
	a := rec.task("a")
	fail := rec.failTask("fail")
	c := rec.task("c")

	cfg := &Config{Auto: Serial(a, fail, c)}
	plan, err := newPublicPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := e2eCtx(t, plan)
	ctx = context.WithValue(ctx, ctxkey.AutoExec{}, true)
	err = cfg.Auto.run(ctx)
	if err == nil {
		t.Fatal("expected error from failing task")
	}

	// A and fail should have records, but C should not.
	names := map[string]bool{}
	for _, r := range rec.records {
		names[r.TaskName] = true
	}
	if !names["a"] {
		t.Error("expected task a to have run")
	}
	if !names["fail"] {
		t.Error("expected task fail to have run")
	}
	if names["c"] {
		t.Error("task c should NOT have run after failure")
	}
}

func TestE2E_AutoExec_ManualSkipped(t *testing.T) {
	e2eSetup(t)
	rec := newRecorder()
	autoTask := rec.task("auto-task")
	manualTask := rec.task("manual-task")

	cfg := &Config{
		Auto:   autoTask,
		Manual: []Runnable{manualTask},
	}
	plan, err := newPublicPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := e2eCtx(t, plan)
	ctx = context.WithValue(ctx, ctxkey.AutoExec{}, true)

	// Run auto tree.
	if err := cfg.Auto.run(ctx); err != nil {
		t.Fatal(err)
	}
	// Attempt to run manual task in auto-exec context.
	if err := manualTask.run(ctx); err != nil {
		t.Fatal(err)
	}

	// Only auto-task should have recorded.
	names := map[string]bool{}
	for _, r := range rec.records {
		names[r.TaskName] = true
	}
	if !names["auto-task"] {
		t.Error("auto-task should have run")
	}
	if names["manual-task"] {
		t.Error("manual-task should be skipped in auto exec mode")
	}
}

func TestE2E_AutoExec_Deduplication(t *testing.T) {
	e2eSetup(t)
	rec := newRecorder()
	shared := rec.task("shared")

	cfg := &Config{Auto: Serial(shared, Parallel(shared, shared))}
	plan, err := newPublicPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := e2eCtx(t, plan)
	ctx = context.WithValue(ctx, ctxkey.AutoExec{}, true)
	if err := cfg.Auto.run(ctx); err != nil {
		t.Fatal(err)
	}

	if len(rec.records) != 1 {
		t.Errorf("expected 1 record (deduplicated), got %d", len(rec.records))
	}
}

func TestE2E_AutoExec_PathFilterWithDetect(t *testing.T) {
	tmpDir := e2eSetup(t)

	// Create dirs with marker files.
	for _, d := range []string{"svc-a", "svc-b", "lib-c"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Only svc-a and svc-b have go.mod.
	for _, d := range []string{"svc-a", "svc-b"} {
		if err := os.WriteFile(
			filepath.Join(tmpDir, d, "go.mod"),
			[]byte("module "+d),
			0o644,
		); err != nil {
			t.Fatal(err)
		}
	}

	rec := newRecorder()
	task := rec.task("detected")

	cfg := &Config{
		Auto: WithOptions(task, WithDetect(DetectByFile("go.mod"))),
	}
	plan, err := newPublicPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx := e2eCtx(t, plan)
	ctx = context.WithValue(ctx, ctxkey.AutoExec{}, true)
	if err := cfg.Auto.run(ctx); err != nil {
		t.Fatal(err)
	}

	if len(rec.records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(rec.records))
	}
	paths := map[string]bool{}
	for _, r := range rec.records {
		paths[r.Path] = true
	}
	if !paths["svc-a"] || !paths["svc-b"] {
		t.Errorf("expected paths svc-a and svc-b, got %v", paths)
	}
}

func TestE2E_AutoExec_TaskScope(t *testing.T) {
	tmpDir := e2eSetup(t)

	for _, d := range []string{"svc-a", "svc-b"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	rec := newRecorder()
	rootTask := rec.task("root")
	scopedTask := rec.task("scoped")

	cfg := &Config{
		Auto: Serial(
			rootTask,
			WithOptions(scopedTask, WithPath("svc-a", "svc-b")),
		),
	}
	plan, err := newPublicPlan(cfg)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("TASK_SCOPE", "svc-a")

	ctx := e2eCtx(t, plan)
	ctx = context.WithValue(ctx, ctxkey.AutoExec{}, true)
	if err := cfg.Auto.run(ctx); err != nil {
		t.Fatal(err)
	}

	want := []execRecord{{TaskName: "scoped", Path: "svc-a"}}
	assert.DeepEqual(t, rec.records, want)
}
