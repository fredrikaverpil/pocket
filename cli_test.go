package pocket

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestPrintTaskHelp_NoArgs(t *testing.T) {
	task := &Task{
		Name:  "test-task",
		Usage: "a test task",
	}

	// Verify task with no args is set up correctly
	if len(task.Args) != 0 {
		t.Error("expected no args")
	}
}

func TestPrintTaskHelp_WithArgs(t *testing.T) {
	task := &Task{
		Name:  "greet",
		Usage: "print a greeting",
		Args: []ArgDef{
			{Name: "name", Usage: "who to greet", Default: "world"},
			{Name: "count", Usage: "how many times"},
		},
	}

	if len(task.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(task.Args))
	}
	if task.Args[0].Name != "name" {
		t.Errorf("expected first arg name='name', got %q", task.Args[0].Name)
	}
	if task.Args[0].Default != "world" {
		t.Errorf("expected first arg default='world', got %q", task.Args[0].Default)
	}
	if task.Args[1].Default != "" {
		t.Errorf("expected second arg no default, got %q", task.Args[1].Default)
	}
}

func TestParseKeyValue(t *testing.T) {
	tests := []struct {
		input   string
		wantKey string
		wantVal string
		wantOK  bool
	}{
		{"name=world", "name", "world", true},
		{"count=10", "count", "10", true},
		{"empty=", "empty", "", true},
		{"with=equals=sign", "with", "equals=sign", true},
		{"noequals", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			key, val, ok := strings.Cut(tt.input, "=")
			if ok != tt.wantOK {
				t.Errorf("Cut(%q): got ok=%v, want %v", tt.input, ok, tt.wantOK)
			}
			if ok {
				if key != tt.wantKey {
					t.Errorf("Cut(%q): got key=%q, want %q", tt.input, key, tt.wantKey)
				}
				if val != tt.wantVal {
					t.Errorf("Cut(%q): got val=%q, want %q", tt.input, val, tt.wantVal)
				}
			}
		})
	}
}

func TestDetectCwd_WithEnvVar(t *testing.T) {
	// Set the environment variable.
	os.Setenv("POK_CONTEXT", "proj1")
	defer os.Unsetenv("POK_CONTEXT")

	cwd := detectCwd()
	if cwd != "proj1" {
		t.Errorf("expected cwd to be 'proj1', got %q", cwd)
	}
}

func TestDetectCwd_WithoutEnvVar(t *testing.T) {
	// Ensure the environment variable is not set.
	os.Unsetenv("POK_CONTEXT")

	cwd := detectCwd()
	// Should fall back to detecting from actual cwd.
	// Since we're running in the repo, it should return "." or a valid path.
	if cwd == "" {
		t.Error("expected cwd to be non-empty")
	}
}

func TestFilterTasksByCwd(t *testing.T) {
	task1 := &Task{Name: "task1"}
	task2 := &Task{Name: "task2"}
	task3 := &Task{Name: "task3"} // no path mapping

	// Create path mappings.
	// task1 runs in proj1, task2 runs in root.
	mappings := map[string]*Paths{
		"task1": P(&cliMockRunnable{}).In("proj1"),
		"task2": P(&cliMockRunnable{}).In("."),
	}

	tasks := []*Task{task1, task2, task3}

	// Test filtering from root.
	rootTasks := filterTasksByCwd(tasks, ".", mappings)
	if len(rootTasks) != 2 {
		t.Errorf("expected 2 tasks at root, got %d", len(rootTasks))
	}
	// task2 and task3 should be visible (task2 has ".", task3 has no mapping but root-only).

	// Test filtering from proj1.
	proj1Tasks := filterTasksByCwd(tasks, "proj1", mappings)
	if len(proj1Tasks) != 1 {
		t.Errorf("expected 1 task in proj1, got %d", len(proj1Tasks))
	}
	if proj1Tasks[0].Name != "task1" {
		t.Errorf("expected task1 in proj1, got %s", proj1Tasks[0].Name)
	}

	// Test filtering from unknown directory.
	otherTasks := filterTasksByCwd(tasks, "other", mappings)
	if len(otherTasks) != 0 {
		t.Errorf("expected 0 tasks in other, got %d", len(otherTasks))
	}
}

func TestIsTaskVisibleIn(t *testing.T) {
	mappings := map[string]*Paths{
		"task1": P(&cliMockRunnable{}).In("proj1", "proj2"),
		"task2": P(&cliMockRunnable{}).In("."),
	}

	tests := []struct {
		taskName string
		cwd      string
		visible  bool
	}{
		{"task1", "proj1", true},
		{"task1", "proj2", true},
		{"task1", ".", false},
		{"task1", "other", false},
		{"task2", ".", true},
		{"task2", "proj1", false},
		{"task3", ".", true}, // no mapping = root only
		{"task3", "proj1", false},
	}

	for _, tt := range tests {
		result := isTaskVisibleIn(tt.taskName, tt.cwd, mappings)
		if result != tt.visible {
			t.Errorf("isTaskVisibleIn(%q, %q) = %v, want %v",
				tt.taskName, tt.cwd, result, tt.visible)
		}
	}
}

// cliMockRunnable is a minimal Runnable for CLI tests.
type cliMockRunnable struct {
	tasks []*Task
}

func (m *cliMockRunnable) Run(_ context.Context) error {
	return nil
}

func (m *cliMockRunnable) Tasks() []*Task {
	return m.tasks
}
