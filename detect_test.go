package pocket

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func TestDetectByFile(t *testing.T) {
	// Not parallel due to shared gitRoot variable.

	tests := []struct {
		name      string
		files     map[string]string // path -> content
		markers   []string          // filenames to detect
		wantPaths []string
	}{
		{
			name: "single go.mod at root",
			files: map[string]string{
				"go.mod": "module example",
			},
			markers:   []string{"go.mod"},
			wantPaths: []string{"."},
		},
		{
			name: "go.mod in subdirectory",
			files: map[string]string{
				"go.mod":         "module example",
				"subdir/go.mod":  "module example/subdir",
				"another/go.mod": "module example/another",
			},
			markers:   []string{"go.mod"},
			wantPaths: []string{".", "another", "subdir"},
		},
		{
			name: "multiple marker files",
			files: map[string]string{
				"pyproject.toml":  "[project]",
				"legacy/setup.py": "from setuptools import setup",
			},
			markers:   []string{"pyproject.toml", "setup.py"},
			wantPaths: []string{".", "legacy"},
		},
		{
			name: "skips hidden directories",
			files: map[string]string{
				"go.mod":         "module example",
				".hidden/go.mod": "module hidden",
			},
			markers:   []string{"go.mod"},
			wantPaths: []string{"."},
		},
		{
			name: "skips vendor directory",
			files: map[string]string{
				"go.mod":        "module example",
				"vendor/go.mod": "module vendor",
			},
			markers:   []string{"go.mod"},
			wantPaths: []string{"."},
		},
		{
			name: "skips node_modules directory",
			files: map[string]string{
				"package.json":              "{}",
				"node_modules/package.json": "{}",
			},
			markers:   []string{"package.json"},
			wantPaths: []string{"."},
		},
		{
			name: "deeply nested",
			files: map[string]string{
				"go.mod":       "module example",
				"a/b/c/go.mod": "module example/a/b/c",
			},
			markers:   []string{"go.mod"},
			wantPaths: []string{".", "a/b/c"},
		},
		{
			name: "deduplicates directories with multiple markers",
			files: map[string]string{
				"pyproject.toml": "[project]",
				"setup.py":       "from setuptools import setup",
			},
			markers:   []string{"pyproject.toml", "setup.py"},
			wantPaths: []string{"."}, // Should only appear once.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory with test files.
			tmpDir := t.TempDir()

			for path, content := range tt.files {
				fullPath := filepath.Join(tmpDir, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
					t.Fatalf("creating directory: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
					t.Fatalf("writing file: %v", err)
				}
			}

			// Override git root for this test.
			origRoot := gitRoot
			gitRoot = tmpDir
			defer func() { gitRoot = origRoot }()

			got := DetectByFile(tt.markers...)

			if !reflect.DeepEqual(got, tt.wantPaths) {
				t.Errorf("DetectByFile(%v) = %v, want %v", tt.markers, got, tt.wantPaths)
			}
		})
	}
}

func TestTaskGroupDef_Integration(t *testing.T) {
	// Test that TaskGroupDef correctly feeds detected modules to the config.
	// Not parallel due to shared gitRoot variable.

	tmpDir := t.TempDir()

	// Create test directory structure with go.mod files.
	files := map[string]string{
		"go.mod":              "module example\n\ngo 1.21",
		"services/api/go.mod": "module example/services/api\n\ngo 1.21",
		"libs/common/go.mod":  "module example/libs/common\n\ngo 1.21",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("creating directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("writing file: %v", err)
		}
	}

	// Override git root.
	origRoot := gitRoot
	gitRoot = tmpDir
	defer func() { gitRoot = origRoot }()

	// Create a TaskGroupDef and use Auto() to create a task group.
	def := TaskGroupDef[testOptions]{
		Name:   "test",
		Detect: func() []string { return DetectByFile("go.mod") },
		Tasks: []TaskDef[testOptions]{
			{Name: "test-task", Create: func(_ map[string]testOptions) *Task {
				return &Task{Name: "test-task", Usage: "test"}
			}},
		},
	}
	tg := def.Auto(testOptions{})

	// Verify modules are detected.
	modules := tg.ModuleConfigs()
	if len(modules) != 3 {
		t.Errorf("expected 3 modules, got %d: %v", len(modules), modules)
	}

	expectedPaths := []string{".", "libs/common", "services/api"}
	for _, path := range expectedPaths {
		if _, ok := modules[path]; !ok {
			t.Errorf("expected module %q not found", path)
		}
	}

	// Test AllModulePaths works with auto-detection.
	cfg := Config{
		TaskGroups: []TaskGroup{tg},
	}
	paths := AllModulePaths(cfg)
	if len(paths) != 3 {
		t.Errorf("AllModulePaths: expected 3 paths, got %d: %v", len(paths), paths)
	}
}

// testOptions is a simple Options type for testing.
type testOptions struct {
	Skip []string
}

func (o testOptions) ShouldRun(taskName string) bool {
	return !slices.Contains(o.Skip, taskName)
}

func TestDetectByExtension(t *testing.T) {
	// Not parallel due to shared gitRoot variable.

	tests := []struct {
		name       string
		files      map[string]string // path -> content
		extensions []string
		wantPaths  []string
	}{
		{
			name: "single lua file at root",
			files: map[string]string{
				"init.lua": "-- lua",
			},
			extensions: []string{".lua"},
			wantPaths:  []string{"."},
		},
		{
			name: "lua files in multiple directories",
			files: map[string]string{
				"init.lua":         "-- lua",
				"scripts/util.lua": "-- lua",
			},
			extensions: []string{".lua"},
			wantPaths:  []string{".", "scripts"},
		},
		{
			name: "multiple extensions",
			files: map[string]string{
				"main.lua":  "-- lua",
				"test.luau": "-- luau",
			},
			extensions: []string{".lua", ".luau"},
			wantPaths:  []string{"."},
		},
		{
			name: "skips hidden directories",
			files: map[string]string{
				"init.lua":         "-- lua",
				".config/init.lua": "-- lua",
			},
			extensions: []string{".lua"},
			wantPaths:  []string{"."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for path, content := range tt.files {
				fullPath := filepath.Join(tmpDir, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
					t.Fatalf("creating directory: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
					t.Fatalf("writing file: %v", err)
				}
			}

			origRoot := gitRoot
			gitRoot = tmpDir
			defer func() { gitRoot = origRoot }()

			got := DetectByExtension(tt.extensions...)

			if !reflect.DeepEqual(got, tt.wantPaths) {
				t.Errorf("DetectByExtension(%v) = %v, want %v", tt.extensions, got, tt.wantPaths)
			}
		})
	}
}
