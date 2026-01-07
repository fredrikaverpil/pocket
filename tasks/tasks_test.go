package tasks_test

import (
	"testing"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tasks"
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
	if t.Go != nil {
		goyek.Undefine(t.Go.Format)
		goyek.Undefine(t.Go.Lint)
		goyek.Undefine(t.Go.Test)
		goyek.Undefine(t.Go.Vulncheck)
	}
	if t.Lua != nil {
		goyek.Undefine(t.Lua.Format)
	}
	if t.Markdown != nil {
		goyek.Undefine(t.Markdown.Format)
	}
	for _, custom := range t.Custom {
		goyek.Undefine(custom)
	}
}

func TestNew_CustomTasks(t *testing.T) {
	customTask := goyek.Task{
		Name:  "my-custom-task",
		Usage: "a custom task for testing",
	}

	cfg := pocket.Config{
		Custom: map[string][]goyek.Task{
			".": {customTask},
		},
	}

	result := tasks.New(cfg)
	defer undefineTasks(result)

	// Verify custom task is registered.
	if len(result.Custom) != 1 {
		t.Fatalf("expected 1 custom task, got %d", len(result.Custom))
	}
	if result.Custom[0].Name() != "my-custom-task" {
		t.Errorf("expected custom task name 'my-custom-task', got %q", result.Custom[0].Name())
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
		Custom: map[string][]goyek.Task{
			".": {
				{Name: "deploy", Usage: "deploy the app"},
				{Name: "release", Usage: "create a release"},
			},
		},
	}

	result := tasks.New(cfg)
	defer undefineTasks(result)

	if len(result.Custom) != 2 {
		t.Fatalf("expected 2 custom tasks, got %d", len(result.Custom))
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

func TestNew_GoTasksConfigDriven(t *testing.T) {
	tests := []struct {
		name          string
		cfg           pocket.Config
		wantInDeps    []string
		wantNotInDeps []string
	}{
		{
			name: "all Go tasks enabled",
			cfg: pocket.Config{
				Go: &pocket.GoConfig{
					Modules: map[string]pocket.GoModuleOptions{
						".": {},
					},
				},
			},
			wantInDeps:    []string{"go-format", "go-lint", "go-test", "go-vulncheck"},
			wantNotInDeps: nil,
		},
		{
			name: "skip format excludes go-format from deps",
			cfg: pocket.Config{
				Go: &pocket.GoConfig{
					Modules: map[string]pocket.GoModuleOptions{
						".": {SkipFormat: true},
					},
				},
			},
			wantInDeps:    []string{"go-lint", "go-test", "go-vulncheck"},
			wantNotInDeps: []string{"go-format"},
		},
		{
			name: "skip all excludes all Go tasks from deps",
			cfg: pocket.Config{
				Go: &pocket.GoConfig{
					Modules: map[string]pocket.GoModuleOptions{
						".": {
							SkipFormat:    true,
							SkipLint:      true,
							SkipTest:      true,
							SkipVulncheck: true,
						},
					},
				},
			},
			wantInDeps:    []string{"generate"},
			wantNotInDeps: []string{"go-format", "go-lint", "go-test", "go-vulncheck"},
		},
		{
			name: "multiple modules with mixed skips",
			cfg: pocket.Config{
				Go: &pocket.GoConfig{
					Modules: map[string]pocket.GoModuleOptions{
						".":      {SkipFormat: true, SkipLint: true, SkipTest: true, SkipVulncheck: true},
						"subdir": {}, // This module has all tasks enabled.
					},
				},
			},
			wantInDeps:    []string{"go-format", "go-lint", "go-test", "go-vulncheck"},
			wantNotInDeps: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tasks.New(tt.cfg)
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

func TestNew_LuaTasksConfigDriven(t *testing.T) {
	tests := []struct {
		name          string
		cfg           pocket.Config
		wantLuaFormat bool
	}{
		{
			name: "lua format enabled",
			cfg: pocket.Config{
				Lua: &pocket.LuaConfig{
					Modules: map[string]pocket.LuaModuleOptions{
						".": {},
					},
				},
			},
			wantLuaFormat: true,
		},
		{
			name: "lua format skipped",
			cfg: pocket.Config{
				Lua: &pocket.LuaConfig{
					Modules: map[string]pocket.LuaModuleOptions{
						".": {SkipFormat: true},
					},
				},
			},
			wantLuaFormat: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tasks.New(tt.cfg)
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

func TestNew_MarkdownTasksConfigDriven(t *testing.T) {
	tests := []struct {
		name         string
		cfg          pocket.Config
		wantMdFormat bool
	}{
		{
			name: "markdown format enabled",
			cfg: pocket.Config{
				Markdown: &pocket.MarkdownConfig{
					Modules: map[string]pocket.MarkdownModuleOptions{
						".": {},
					},
				},
			},
			wantMdFormat: true,
		},
		{
			name: "markdown format skipped",
			cfg: pocket.Config{
				Markdown: &pocket.MarkdownConfig{
					Modules: map[string]pocket.MarkdownModuleOptions{
						".": {SkipFormat: true},
					},
				},
			},
			wantMdFormat: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tasks.New(tt.cfg)
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
	result := tasks.New(pocket.Config{})
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

func TestNew_NoEcosystemsConfigured(t *testing.T) {
	result := tasks.New(pocket.Config{})
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

	// Ecosystem-specific task holders should be nil.
	if result.Go != nil {
		t.Error("Go tasks should be nil when not configured")
	}
	if result.Lua != nil {
		t.Error("Lua tasks should be nil when not configured")
	}
	if result.Markdown != nil {
		t.Error("Markdown tasks should be nil when not configured")
	}
}
