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

func TestParseTaskArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantArgs    map[string]string
		wantHelp    bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "empty args",
			args:     []string{},
			wantArgs: map[string]string{},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "-key=value format",
			args:     []string{"-name=world"},
			wantArgs: map[string]string{"name": "world"},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "-key value format",
			args:     []string{"-name", "world"},
			wantArgs: map[string]string{"name": "world"},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "multiple args with mixed formats",
			args:     []string{"-name=Claude", "-count", "10"},
			wantArgs: map[string]string{"name": "Claude", "count": "10"},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "value with equals sign",
			args:     []string{"-filter=key=value"},
			wantArgs: map[string]string{"filter": "key=value"},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "empty value",
			args:     []string{"-empty="},
			wantArgs: map[string]string{"empty": ""},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "value with spaces",
			args:     []string{"-msg=hello world"},
			wantArgs: map[string]string{"msg": "hello world"},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "value with spaces using space separator",
			args:     []string{"-msg", "hello world"},
			wantArgs: map[string]string{"msg": "hello world"},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "help flag",
			args:     []string{"-h"},
			wantArgs: nil,
			wantHelp: true,
			wantErr:  false,
		},
		{
			name:     "help flag after args",
			args:     []string{"-name=world", "-h"},
			wantArgs: nil,
			wantHelp: true,
			wantErr:  false,
		},
		{
			name:        "missing dash prefix",
			args:        []string{"name=world"},
			wantArgs:    nil,
			wantHelp:    false,
			wantErr:     true,
			errContains: "expected -key=value or -key value",
		},
		{
			name:        "missing value for space format",
			args:        []string{"-name"},
			wantArgs:    nil,
			wantHelp:    false,
			wantErr:     true,
			errContains: "missing value for argument -name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArgs, gotHelp, err := parseTaskArgs(tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseTaskArgs(%v): expected error containing %q, got nil", tt.args, tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parseTaskArgs(%v): error %q does not contain %q", tt.args, err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("parseTaskArgs(%v): unexpected error: %v", tt.args, err)
				return
			}

			if gotHelp != tt.wantHelp {
				t.Errorf("parseTaskArgs(%v): got help=%v, want %v", tt.args, gotHelp, tt.wantHelp)
			}

			if tt.wantHelp {
				return
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("parseTaskArgs(%v): got %d args, want %d", tt.args, len(gotArgs), len(tt.wantArgs))
			}

			for k, wantV := range tt.wantArgs {
				if gotV, ok := gotArgs[k]; !ok {
					t.Errorf("parseTaskArgs(%v): missing key %q", tt.args, k)
				} else if gotV != wantV {
					t.Errorf("parseTaskArgs(%v): key %q = %q, want %q", tt.args, k, gotV, wantV)
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
	mappings := map[string]*PathFilter{
		"task1": Paths(&cliMockRunnable{}).In("proj1"),
		"task2": Paths(&cliMockRunnable{}).In("."),
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
	mappings := map[string]*PathFilter{
		"task1": Paths(&cliMockRunnable{}).In("proj1", "proj2"),
		"task2": Paths(&cliMockRunnable{}).In("."),
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
