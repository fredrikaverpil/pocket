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

func TestBuildPlanFromJSONBytes(t *testing.T) {
	task := &Task{Name: "test", Usage: "test", Do: func(_ context.Context) error { return nil }}
	_ = task.buildFlagSet()
	cfg := &Config{Auto: Serial(task)}
	basePlan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	data := []byte(`{"version":1,"tree":{"type":"task","name":"test"}}`)
	plan, err := buildPlanFromJSONBytes(data, basePlan)
	if err != nil {
		t.Fatal(err)
	}
	if plan.tree == nil {
		t.Fatal("expected JSON plan tree")
	}
	if inst := plan.taskInstanceByName("test"); inst == nil {
		t.Fatal("expected task instance from JSON plan")
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
