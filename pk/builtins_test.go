package pk

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/fredrikaverpil/pocket/pk/internal/ctxkey"
	pkrun "github.com/fredrikaverpil/pocket/pk/run"
)

func TestGitDiffTask_Disabled(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxkey.GitDiff{}, false)
	ctx = context.WithValue(ctx, ctxkey.Output{}, &pkrun.Output{Stdout: io.Discard, Stderr: io.Discard})

	// Should return nil immediately when git diff is disabled
	if err := gitDiffTask.run(ctx); err != nil {
		t.Errorf("gitDiffTask.run() with disabled flag returned error: %v", err)
	}
}

func TestGitDiffEnabledFromContext_Default(t *testing.T) {
	ctx := context.Background()

	// Default should be false (git diff disabled)
	if gitDiffEnabled(ctx) {
		t.Error("expected gitDiffEnabled to be false by default")
	}
}

func TestGitDiffEnabledFromContext_Enabled(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxkey.GitDiff{}, true)

	if !gitDiffEnabled(ctx) {
		t.Error("expected gitDiffEnabled to be true after setting")
	}
}

func TestIsBuiltinName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"shims", true},
		{"plan", true},
		{"exec", true},
		{"git-diff", true},
		{"self-update", true},
		{"purge", true},
		{"commits-check", true},
		{"lint", false},
		{"", false},
		{"Plan", false}, // case-sensitive
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isBuiltinName(tc.name); got != tc.want {
				t.Errorf("isBuiltinName(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestCommitsCheckTask_Disabled(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxkey.CommitsCheck{}, false)
	ctx = context.WithValue(ctx, ctxkey.Output{}, &pkrun.Output{Stdout: io.Discard, Stderr: io.Discard})

	// Should return nil immediately when commits check is disabled.
	if err := commitsCheckTask.run(ctx); err != nil {
		t.Errorf("commitsCheckTask.run() with disabled flag returned error: %v", err)
	}
}

func TestCommitsCheckEnabledFromContext_Default(t *testing.T) {
	ctx := context.Background()

	// Default should be false (commits check disabled).
	if commitsCheckEnabled(ctx) {
		t.Error("expected commitsCheckEnabled to be false by default")
	}
}

func TestCommitsCheckEnabledFromContext_Enabled(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxkey.CommitsCheck{}, true)

	if !commitsCheckEnabled(ctx) {
		t.Error("expected commitsCheckEnabled to be true after setting")
	}
}

func TestFormatPaths(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{"empty", nil, "[root]"},
		{"root dot", []string{"."}, "[root]"},
		{"single path", []string{"services"}, "[services]"},
		{"two paths", []string{"a", "b"}, "[a b]"},
		{"three paths", []string{"a", "b", "c"}, "[a b c]"},
		{"four paths", []string{"a", "b", "c", "d"}, "4 directories"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatPaths(tc.paths)
			if got != tc.want {
				t.Errorf("formatPaths(%v) = %q, want %q", tc.paths, got, tc.want)
			}
		})
	}
}

func TestBuildJSONTree(t *testing.T) {
	task := &Task{Name: "test", Usage: "test", Do: func(_ context.Context) error { return nil }}
	_ = task.buildFlagSet()

	plan := &Plan{
		taskIndex: map[string]*taskInstance{
			"test": {task: task, name: "test"},
		},
		pathMappings: map[string]pathInfo{
			"test": {resolvedPaths: []string{"."}},
		},
	}

	t.Run("SingleTask", func(t *testing.T) {
		result := buildJSONTree(task, "", plan)
		if result["type"] != "task" {
			t.Errorf("expected type=task, got %v", result["type"])
		}
		if result["name"] != "test" {
			t.Errorf("expected name=test, got %v", result["name"])
		}
	})

	t.Run("Serial", func(t *testing.T) {
		s := Serial(task)
		result := buildJSONTree(s, "", plan)
		if result["type"] != "serial" {
			t.Errorf("expected type=serial, got %v", result["type"])
		}
		children := result["children"].([]map[string]any)
		if len(children) != 1 {
			t.Errorf("expected 1 child, got %d", len(children))
		}
	})

	t.Run("Parallel", func(t *testing.T) {
		p := Parallel(task)
		result := buildJSONTree(p, "", plan)
		if result["type"] != "parallel" {
			t.Errorf("expected type=parallel, got %v", result["type"])
		}
	})

	t.Run("Nil", func(t *testing.T) {
		result := buildJSONTree(nil, "", plan)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("WithOptions", func(t *testing.T) {
		pf := WithOptions(task, WithPath("services"))
		result := buildJSONTree(pf, "", plan)
		if result["type"] != "pathFilter" {
			t.Errorf("expected type=pathFilter, got %v", result["type"])
		}
		include := result["include"].([]string)
		if len(include) != 1 || include[0] != "services" {
			t.Errorf("expected include=[services], got %v", include)
		}
	})

	t.Run("WithNameSuffix", func(t *testing.T) {
		// Update plan to include suffixed task.
		plan.taskIndex["test:v2"] = &taskInstance{task: task, name: "test:v2"}
		plan.pathMappings["test:v2"] = pathInfo{resolvedPaths: []string{"."}}

		pf := WithOptions(task, WithNameSuffix("v2"))
		result := buildJSONTree(pf, "", plan)
		// No path options, so pathFilter wrapper is omitted; the inner task is returned directly.
		if result["name"] != "test:v2" {
			t.Errorf("expected name=test:v2, got %v", result["name"])
		}
	})
}

type lintHelpFlags struct {
	Fix bool `flag:"fix" usage:"apply fixes"`
}

func TestPrintTaskHelp(t *testing.T) {
	task := &Task{
		Name:  "lint",
		Usage: "run linters",
		Flags: lintHelpFlags{Fix: true},
	}
	_ = task.buildFlagSet()

	var buf bytes.Buffer
	out := &pkrun.Output{Stdout: &buf, Stderr: &buf}
	ctx := context.WithValue(context.Background(), ctxkey.Output{}, out)

	printTaskHelp(ctx, task)

	output := buf.String()
	for _, want := range []string{"lint", "run linters", "Usage:", "Flags:", "fix"} {
		if !bytes.Contains([]byte(output), []byte(want)) {
			t.Errorf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}

func TestPrintTaskHelp_NoFlags(t *testing.T) {
	task := &Task{
		Name:  "noop",
		Usage: "does nothing",
	}
	_ = task.buildFlagSet()

	var buf bytes.Buffer
	out := &pkrun.Output{Stdout: &buf, Stderr: &buf}
	ctx := context.WithValue(context.Background(), ctxkey.Output{}, out)

	printTaskHelp(ctx, task)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("no flags")) {
		t.Errorf("expected 'no flags' message, got:\n%s", output)
	}
}

func TestPrintHelp(t *testing.T) {
	task := &Task{Name: "lint", Usage: "run linters", Do: func(_ context.Context) error { return nil }}
	cfg := &Config{Auto: Serial(task)}
	plan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	out := &pkrun.Output{Stdout: &buf, Stderr: &buf}
	ctx := context.WithValue(context.Background(), ctxkey.Output{}, out)

	printHelp(ctx, cfg, plan)

	output := buf.String()
	for _, want := range []string{"pocket", "Usage:", "Global flags:", "lint", "run linters", "Builtin tasks:"} {
		if !bytes.Contains([]byte(output), []byte(want)) {
			t.Errorf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}

func TestPrintPlanJSON(t *testing.T) {
	task := &Task{Name: "test", Usage: "test", Do: func(_ context.Context) error { return nil }}
	_ = task.buildFlagSet()
	cfg := &Config{Auto: Serial(task)}
	plan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	out := &pkrun.Output{Stdout: &buf, Stderr: &buf}
	ctx := context.WithValue(context.Background(), ctxkey.Output{}, out)

	if err := printPlanJSON(ctx, plan.tree, plan); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	for _, want := range []string{`"version"`, `"tree"`, `"tasks"`, `"serial"`, `"test"`} {
		if !bytes.Contains([]byte(output), []byte(want)) {
			t.Errorf("expected JSON to contain %q, got:\n%s", want, output)
		}
	}
}

func TestPrintTree(t *testing.T) {
	task := &Task{Name: "test", Usage: "test", Do: func(_ context.Context) error { return nil }}
	_ = task.buildFlagSet()
	cfg := &Config{Auto: Serial(task)}
	plan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	out := &pkrun.Output{Stdout: &buf, Stderr: &buf}
	ctx := context.WithValue(context.Background(), ctxkey.Output{}, out)

	printTree(ctx, plan.tree, "", true, "", plan)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("test")) {
		t.Errorf("expected tree output to contain 'test', got:\n%s", output)
	}
	// Should contain tree formatting characters.
	if !bytes.Contains([]byte(output), []byte("─")) {
		t.Errorf("expected tree formatting characters, got:\n%s", output)
	}
}
