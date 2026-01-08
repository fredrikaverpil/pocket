package tasks_test

import (
	"testing"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tasks"
	"github.com/fredrikaverpil/pocket/tasks/golang"
	"github.com/fredrikaverpil/pocket/tasks/lua"
	"github.com/fredrikaverpil/pocket/tasks/markdown"
	"github.com/goyek/goyek/v3"
)

// undefineTasks cleans up all tasks registered by tasks.New().
// This is necessary because goyek uses a global registry.
func undefineTasks(t *tasks.Tasks) {
	if t.All != nil {
		goyek.Undefine(t.All)
	}
	if t.Generate != nil {
		goyek.Undefine(t.Generate)
	}
	if t.Update != nil {
		goyek.Undefine(t.Update)
	}
	if t.GitDiff != nil {
		goyek.Undefine(t.GitDiff)
	}
	for _, task := range t.TaskGroupTasks {
		goyek.Undefine(task)
	}
	for _, task := range t.Tasks {
		goyek.Undefine(task)
	}
}

func TestNew_CustomTasks(t *testing.T) {
	customTask := goyek.Task{
		Name:  "my-custom-task",
		Usage: "a custom task for testing",
	}

	cfg := pocket.Config{
		Tasks: map[string][]goyek.Task{
			".": {customTask},
		},
	}

	result := tasks.New(cfg, ".")
	defer undefineTasks(result)

	// Verify custom task is registered.
	if len(result.Tasks) != 1 {
		t.Fatalf("expected 1 custom task, got %d", len(result.Tasks))
	}
	if result.Tasks[0].Name() != "my-custom-task" {
		t.Errorf("expected custom task name 'my-custom-task', got %q", result.Tasks[0].Name())
	}

	// Verify custom task is in "all" dependencies.
	allDeps := result.All.Deps()
	found := false
	for _, dep := range allDeps {
		if dep.Name() == "my-custom-task" {
			found = true
			break
		}
	}
	if !found {
		t.Error("custom task not found in 'all' task dependencies")
	}
}

func TestNew_MultipleCustomTasks(t *testing.T) {
	cfg := pocket.Config{
		Tasks: map[string][]goyek.Task{
			".": {
				{Name: "deploy", Usage: "deploy the app"},
				{Name: "release", Usage: "create a release"},
			},
		},
	}

	result := tasks.New(cfg, ".")
	defer undefineTasks(result)

	if len(result.Tasks) != 2 {
		t.Fatalf("expected 2 custom tasks, got %d", len(result.Tasks))
	}

	// Verify both are in "all" dependencies.
	allDeps := result.All.Deps()
	names := make(map[string]bool)
	for _, dep := range allDeps {
		names[dep.Name()] = true
	}

	if !names["deploy"] {
		t.Error("'deploy' task not found in 'all' dependencies")
	}
	if !names["release"] {
		t.Error("'release' task not found in 'all' dependencies")
	}
}

func TestNew_GoTaskGroupConfigDriven(t *testing.T) {
	tests := []struct {
		name          string
		taskGroup     pocket.TaskGroup
		wantInDeps    []string
		wantNotInDeps []string
	}{
		{
			name: "all Go tasks enabled",
			taskGroup: golang.New(map[string]golang.Options{
				".": {},
			}),
			wantInDeps:    []string{"go-format", "go-lint", "go-test", "go-vulncheck"},
			wantNotInDeps: nil,
		},
		{
			name: "skip format excludes go-format from deps",
			taskGroup: golang.New(map[string]golang.Options{
				".": {Skip: []string{"format"}},
			}),
			wantInDeps:    []string{"go-lint", "go-test", "go-vulncheck"},
			wantNotInDeps: []string{"go-format"},
		},
		{
			name: "skip all excludes all Go tasks from deps",
			taskGroup: golang.New(map[string]golang.Options{
				".": {Skip: []string{"format", "lint", "test", "vulncheck"}},
			}),
			wantInDeps:    []string{"generate"},
			wantNotInDeps: []string{"go-format", "go-lint", "go-test", "go-vulncheck"},
		},
		{
			name: "multiple modules with mixed skips",
			taskGroup: golang.New(map[string]golang.Options{
				".":      {Skip: []string{"format", "lint", "test", "vulncheck"}},
				"subdir": {}, // This module has all tasks enabled.
			}),
			wantInDeps:    []string{"go-format", "go-lint", "go-test", "go-vulncheck"},
			wantNotInDeps: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := pocket.Config{
				TaskGroups: []pocket.TaskGroup{tt.taskGroup},
			}
			result := tasks.New(cfg, ".")
			defer undefineTasks(result)

			allDeps := result.All.Deps()

			depNames := make(map[string]bool)
			for _, dep := range allDeps {
				depNames[dep.Name()] = true
			}

			for _, want := range tt.wantInDeps {
				if !depNames[want] {
					t.Errorf("expected %q in 'all' dependencies, but not found", want)
				}
			}

			for _, notWant := range tt.wantNotInDeps {
				if depNames[notWant] {
					t.Errorf("expected %q NOT in 'all' dependencies, but found", notWant)
				}
			}
		})
	}
}

func TestNew_LuaTaskGroupConfigDriven(t *testing.T) {
	tests := []struct {
		name          string
		taskGroup     pocket.TaskGroup
		wantLuaFormat bool
	}{
		{
			name: "lua format enabled",
			taskGroup: lua.New(map[string]lua.Options{
				".": {},
			}),
			wantLuaFormat: true,
		},
		{
			name: "lua format skipped",
			taskGroup: lua.New(map[string]lua.Options{
				".": {Skip: []string{"format"}},
			}),
			wantLuaFormat: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := pocket.Config{
				TaskGroups: []pocket.TaskGroup{tt.taskGroup},
			}
			result := tasks.New(cfg, ".")
			defer undefineTasks(result)

			allDeps := result.All.Deps()

			found := false
			for _, dep := range allDeps {
				if dep.Name() == "lua-format" {
					found = true
					break
				}
			}

			if found != tt.wantLuaFormat {
				t.Errorf("lua-format in deps = %v, want %v", found, tt.wantLuaFormat)
			}
		})
	}
}

func TestNew_MarkdownTaskGroupConfigDriven(t *testing.T) {
	tests := []struct {
		name         string
		taskGroup    pocket.TaskGroup
		wantMdFormat bool
	}{
		{
			name: "markdown format enabled",
			taskGroup: markdown.New(map[string]markdown.Options{
				".": {},
			}),
			wantMdFormat: true,
		},
		{
			name: "markdown format skipped",
			taskGroup: markdown.New(map[string]markdown.Options{
				".": {Skip: []string{"format"}},
			}),
			wantMdFormat: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := pocket.Config{
				TaskGroups: []pocket.TaskGroup{tt.taskGroup},
			}
			result := tasks.New(cfg, ".")
			defer undefineTasks(result)

			allDeps := result.All.Deps()

			found := false
			for _, dep := range allDeps {
				if dep.Name() == "md-format" {
					found = true
					break
				}
			}

			if found != tt.wantMdFormat {
				t.Errorf("md-format in deps = %v, want %v", found, tt.wantMdFormat)
			}
		})
	}
}

func TestNew_GenerateAlwaysInDeps(t *testing.T) {
	// Even with empty config, generate should be in deps.
	result := tasks.New(pocket.Config{}, ".")
	defer undefineTasks(result)

	allDeps := result.All.Deps()
	found := false
	for _, dep := range allDeps {
		if dep.Name() == "generate" {
			found = true
			break
		}
	}

	if !found {
		t.Error("'generate' task should always be in 'all' dependencies")
	}
}

func TestNew_NoTaskGroupsRegistered(t *testing.T) {
	result := tasks.New(pocket.Config{}, ".")
	defer undefineTasks(result)

	// Should have Generate, All, Update, GitDiff defined.
	if result.Generate == nil {
		t.Error("Generate task should be defined")
	}
	if result.All == nil {
		t.Error("All task should be defined")
	}
	if result.Update == nil {
		t.Error("Update task should be defined")
	}
	if result.GitDiff == nil {
		t.Error("GitDiff task should be defined")
	}

	// No task group tasks should be registered.
	if len(result.TaskGroupTasks) != 0 {
		t.Errorf("expected 0 task group tasks, got %d", len(result.TaskGroupTasks))
	}
}

func TestNew_ContextFiltering(t *testing.T) {
	// Create a task group with modules in different contexts.
	goTaskGroup := golang.New(map[string]golang.Options{
		".":     {},
		"tests": {},
	})

	cfg := pocket.Config{
		TaskGroups: []pocket.TaskGroup{goTaskGroup},
		Tasks: map[string][]goyek.Task{
			".":      {{Name: "root-task", Usage: "root only"}},
			"tests":  {{Name: "tests-task", Usage: "tests only"}},
			"deploy": {{Name: "deploy-task", Usage: "deploy only"}},
		},
	}

	t.Run("root context includes all", func(t *testing.T) {
		result := tasks.New(cfg, ".")
		defer undefineTasks(result)

		// Should include root custom task and all kit tasks.
		allDeps := result.All.Deps()
		names := make(map[string]bool)
		for _, dep := range allDeps {
			names[dep.Name()] = true
		}

		if !names["root-task"] {
			t.Error("expected root-task in deps for root context")
		}
		// Task group tasks should be present.
		if !names["go-format"] {
			t.Error("expected go-format in deps for root context")
		}
	})

	t.Run("tests context filters to tests only", func(t *testing.T) {
		result := tasks.New(cfg, "tests")
		defer undefineTasks(result)

		allDeps := result.All.Deps()
		names := make(map[string]bool)
		for _, dep := range allDeps {
			names[dep.Name()] = true
		}

		if !names["tests-task"] {
			t.Error("expected tests-task in deps for tests context")
		}
		if names["root-task"] {
			t.Error("did not expect root-task in deps for tests context")
		}
		// Task group should still have tasks since it has a "tests" module.
		if !names["go-format"] {
			t.Error("expected go-format in deps for tests context (task group has tests module)")
		}
	})

	t.Run("deploy context has no task group tasks", func(t *testing.T) {
		result := tasks.New(cfg, "deploy")
		defer undefineTasks(result)

		allDeps := result.All.Deps()
		names := make(map[string]bool)
		for _, dep := range allDeps {
			names[dep.Name()] = true
		}

		if !names["deploy-task"] {
			t.Error("expected deploy-task in deps for deploy context")
		}
		// Task group doesn't have a deploy module, so no task group tasks.
		if names["go-format"] {
			t.Error("did not expect go-format in deps for deploy context (task group has no deploy module)")
		}
	})
}
