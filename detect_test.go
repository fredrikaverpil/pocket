package pocket

import (
	"os"
	"path/filepath"
	"reflect"
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
