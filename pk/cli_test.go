package pk

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/fredrikaverpil/pocket/pk/repopath"
)

func TestRun_ErrorsOutsideGitRepo(t *testing.T) {
	withArgs(t, "pok")
	withWorkingDir(t, t.TempDir())
	repopath.SetGitRootFunc(nil)

	_, err := run(&Config{})
	if !errors.Is(err, repopath.ErrGitRootNotFound) {
		t.Fatalf("expected ErrGitRootNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), "finding git root") {
		t.Errorf("expected git root context, got %v", err)
	}
}

func TestRun_VersionDoesNotRequireGitRepo(t *testing.T) {
	withArgs(t, "pok", "--version")
	withWorkingDir(t, t.TempDir())
	repopath.SetGitRootFunc(nil)

	if _, err := run(&Config{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindTask_Builtin(t *testing.T) {
	// findTask should return builtins even with nil plan.
	instance := findTask(nil, "plan")
	if instance == nil {
		t.Fatal("expected to find builtin 'plan'")
	}
	if instance.task.Name != "plan" {
		t.Errorf("expected task name 'plan', got %q", instance.task.Name)
	}
}

func TestFindTask_UserTask(t *testing.T) {
	task := &Task{Name: "lint", Usage: "lint", Do: func(_ context.Context) error { return nil }}
	cfg := &Config{Auto: Serial(task)}
	plan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	instance := findTask(plan, "lint")
	if instance == nil {
		t.Fatal("expected to find user task 'lint'")
	}
	if instance.name != "lint" {
		t.Errorf("expected name 'lint', got %q", instance.name)
	}
}

func TestFindTask_Unknown(t *testing.T) {
	task := &Task{Name: "lint", Usage: "lint", Do: func(_ context.Context) error { return nil }}
	cfg := &Config{Auto: Serial(task)}
	plan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	instance := findTask(plan, "nonexistent")
	if instance != nil {
		t.Errorf("expected nil for unknown task, got %v", instance)
	}
}

func TestFindTask_NilPlan(t *testing.T) {
	// Builtins should still work with nil plan.
	instance := findTask(nil, "shims")
	if instance == nil {
		t.Fatal("expected to find builtin 'shims' even with nil plan")
	}

	// Non-builtins should return nil.
	instance = findTask(nil, "lint")
	if instance != nil {
		t.Error("expected nil for non-builtin with nil plan")
	}
}

type pyCliFlags struct {
	Python string `flag:"python" usage:"python version"`
}

func TestFindTaskByName_WithSuffix(t *testing.T) {
	task := &Task{
		Name:  "py-test",
		Usage: "python test",
		Flags: pyCliFlags{Python: "3.9"},
		Do:    func(_ context.Context) error { return nil },
	}

	cfg := &Config{
		Auto: Serial(
			WithOptions(task, WithNameSuffix("3.9"), WithFlags(pyCliFlags{Python: "3.9"})),
			WithOptions(task, WithNameSuffix("3.10"), WithFlags(pyCliFlags{Python: "3.10"})),
		),
	}

	plan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	// Look up by suffixed name.
	instance := findTaskByName(plan, "py-test:3.9")
	if instance == nil {
		t.Fatal("expected to find py-test:3.9")
	}
	if instance.name != "py-test:3.9" {
		t.Errorf("expected name 'py-test:3.9', got %q", instance.name)
	}

	instance = findTaskByName(plan, "py-test:3.10")
	if instance == nil {
		t.Fatal("expected to find py-test:3.10")
	}

	// Base name without suffix should not match.
	instance = findTaskByName(plan, "py-test")
	if instance != nil {
		t.Error("expected nil for base name without suffix")
	}
}

func withArgs(t *testing.T, args ...string) {
	t.Helper()

	oldArgs := os.Args
	os.Args = args
	t.Cleanup(func() {
		os.Args = oldArgs
	})
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
		repopath.SetGitRootFunc(nil)
	})
}
