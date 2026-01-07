package shim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fredrikaverpil/bld"
	"github.com/goyek/goyek/v3"
)

func TestCalculateBldDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		context string
		want    string
	}{
		{
			name:    "root context",
			context: ".",
			want:    ".bld",
		},
		{
			name:    "single depth",
			context: "tests",
			want:    filepath.Join("..", ".bld"),
		},
		{
			name:    "two levels deep",
			context: filepath.Join("tests", "integration"),
			want:    filepath.Join("..", "..", ".bld"),
		},
		{
			name:    "three levels deep",
			context: filepath.Join("a", "b", "c"),
			want:    filepath.Join("..", "..", "..", ".bld"),
		},
		{
			name:    "single directory with different name",
			context: "submodule",
			want:    filepath.Join("..", ".bld"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := calculateBldDir(tt.context)
			if got != tt.want {
				t.Errorf("calculateBldDir(%q) = %q, want %q", tt.context, got, tt.want)
			}
		})
	}
}

func TestGenerateWithRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       bld.Config
		wantShims    []string          // Relative paths to expected shim files.
		wantContexts map[string]string // Map of shim path to expected BLD_CONTEXT value.
		wantBldDirs  map[string]string // Map of shim path to expected BLD_DIR value.
	}{
		{
			name: "root only config",
			config: bld.Config{
				Go: &bld.GoConfig{
					Modules: map[string]bld.GoModuleOptions{
						".": {},
					},
				},
			},
			wantShims: []string{"bld"},
			wantContexts: map[string]string{
				"bld": ".",
			},
			wantBldDirs: map[string]string{
				"bld": ".bld",
			},
		},
		{
			name: "root and submodule",
			config: bld.Config{
				Go: &bld.GoConfig{
					Modules: map[string]bld.GoModuleOptions{
						".":     {},
						"tests": {SkipLint: true},
					},
				},
			},
			wantShims: []string{
				"bld",
				filepath.Join("tests", "bld"),
			},
			wantContexts: map[string]string{
				"bld":                         ".",
				filepath.Join("tests", "bld"): "tests",
			},
			wantBldDirs: map[string]string{
				"bld":                         ".bld",
				filepath.Join("tests", "bld"): filepath.Join("..", ".bld"),
			},
		},
		{
			name: "multiple language configs",
			config: bld.Config{
				Go: &bld.GoConfig{
					Modules: map[string]bld.GoModuleOptions{
						".": {},
					},
				},
				Lua: &bld.LuaConfig{
					Modules: map[string]bld.LuaModuleOptions{
						"scripts": {},
					},
				},
			},
			wantShims: []string{
				"bld",
				filepath.Join("scripts", "bld"),
			},
			wantContexts: map[string]string{
				"bld":                           ".",
				filepath.Join("scripts", "bld"): "scripts",
			},
			wantBldDirs: map[string]string{
				"bld":                           ".bld",
				filepath.Join("scripts", "bld"): filepath.Join("..", ".bld"),
			},
		},
		{
			name: "custom shim name",
			config: bld.Config{
				ShimName: "build",
				Go: &bld.GoConfig{
					Modules: map[string]bld.GoModuleOptions{
						".": {},
					},
				},
			},
			wantShims: []string{"build"},
			wantContexts: map[string]string{
				"build": ".",
			},
			wantBldDirs: map[string]string{
				"build": ".bld",
			},
		},
		{
			name: "deeply nested context",
			config: bld.Config{
				Go: &bld.GoConfig{
					Modules: map[string]bld.GoModuleOptions{
						".":                          {},
						filepath.Join("a", "b", "c"): {},
					},
				},
			},
			wantShims: []string{
				"bld",
				filepath.Join("a", "b", "c", "bld"),
			},
			wantContexts: map[string]string{
				"bld":                               ".",
				filepath.Join("a", "b", "c", "bld"): filepath.Join("a", "b", "c"),
			},
			wantBldDirs: map[string]string{
				"bld":                               ".bld",
				filepath.Join("a", "b", "c", "bld"): filepath.Join("..", "..", "..", ".bld"),
			},
		},
		{
			name: "custom tasks context",
			config: bld.Config{
				Custom: map[string][]goyek.Task{
					".":      nil, // Root custom tasks.
					"deploy": nil, // Custom tasks in deploy folder.
				},
			},
			wantShims: []string{
				"bld",
				filepath.Join("deploy", "bld"),
			},
			wantContexts: map[string]string{
				"bld":                          ".",
				filepath.Join("deploy", "bld"): "deploy",
			},
			wantBldDirs: map[string]string{
				"bld":                          ".bld",
				filepath.Join("deploy", "bld"): filepath.Join("..", ".bld"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a temporary directory for testing.
			tmpDir := t.TempDir()

			// Create the .bld directory with a minimal go.mod.
			bldDir := filepath.Join(tmpDir, ".bld")
			if err := os.MkdirAll(bldDir, 0o755); err != nil {
				t.Fatalf("creating .bld directory: %v", err)
			}

			goMod := "module bld\n\ngo 1.23.0\n"
			if err := os.WriteFile(filepath.Join(bldDir, "go.mod"), []byte(goMod), 0o644); err != nil {
				t.Fatalf("writing go.mod: %v", err)
			}

			// Generate shims.
			if err := GenerateWithRoot(tt.config, tmpDir); err != nil {
				t.Fatalf("GenerateWithRoot: %v", err)
			}

			// Verify all expected shims exist.
			for _, shimPath := range tt.wantShims {
				fullPath := filepath.Join(tmpDir, shimPath)
				info, err := os.Stat(fullPath)
				if err != nil {
					t.Errorf("expected shim %q not found: %v", shimPath, err)
					continue
				}

				// Verify executable permissions.
				if info.Mode().Perm()&0o100 == 0 {
					t.Errorf("shim %q is not executable", shimPath)
				}

				// Read and verify content.
				content, err := os.ReadFile(fullPath)
				if err != nil {
					t.Errorf("reading shim %q: %v", shimPath, err)
					continue
				}

				contentStr := string(content)

				// Verify BLD_CONTEXT.
				if wantContext, ok := tt.wantContexts[shimPath]; ok {
					expectedContext := `BLD_CONTEXT="` + wantContext + `"`
					if !strings.Contains(contentStr, expectedContext) {
						t.Errorf("shim %q: expected BLD_CONTEXT=%q not found in content", shimPath, wantContext)
					}
				}

				// Verify BLD_DIR.
				if wantBldDir, ok := tt.wantBldDirs[shimPath]; ok {
					expectedBldDir := `BLD_DIR="` + wantBldDir + `"`
					if !strings.Contains(contentStr, expectedBldDir) {
						t.Errorf("shim %q: expected BLD_DIR=%q not found in content", shimPath, wantBldDir)
					}
				}

				// Verify Go version.
				if !strings.Contains(contentStr, `GO_VERSION="1.23.0"`) {
					t.Errorf("shim %q: expected GO_VERSION=1.23.0 not found", shimPath)
				}
			}
		})
	}
}

func TestGenerateWithRoot_MissingGoMod(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .bld directory without go.mod.
	bldDir := filepath.Join(tmpDir, ".bld")
	if err := os.MkdirAll(bldDir, 0o755); err != nil {
		t.Fatalf("creating .bld directory: %v", err)
	}

	config := bld.Config{
		Go: &bld.GoConfig{
			Modules: map[string]bld.GoModuleOptions{
				".": {},
			},
		},
	}

	err := GenerateWithRoot(config, tmpDir)
	if err == nil {
		t.Error("expected error for missing go.mod, got nil")
	}
	if !strings.Contains(err.Error(), "go.mod") {
		t.Errorf("expected error to mention go.mod, got: %v", err)
	}
}

func TestGenerateWithRoot_MissingGoDirective(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .bld directory with go.mod that has no go directive.
	bldDir := filepath.Join(tmpDir, ".bld")
	if err := os.MkdirAll(bldDir, 0o755); err != nil {
		t.Fatalf("creating .bld directory: %v", err)
	}

	goMod := "module bld\n\nrequire example.com/foo v1.0.0\n"
	if err := os.WriteFile(filepath.Join(bldDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	config := bld.Config{
		Go: &bld.GoConfig{
			Modules: map[string]bld.GoModuleOptions{
				".": {},
			},
		},
	}

	err := GenerateWithRoot(config, tmpDir)
	if err == nil {
		t.Error("expected error for missing go directive, got nil")
	}
	if !strings.Contains(err.Error(), "go directive") {
		t.Errorf("expected error to mention go directive, got: %v", err)
	}
}

func TestExtractGoVersionFromDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		goModContent string
		wantVersion  string
		wantErr      bool
	}{
		{
			name:         "standard go.mod",
			goModContent: "module example\n\ngo 1.23.0\n",
			wantVersion:  "1.23.0",
			wantErr:      false,
		},
		{
			name:         "go directive with extra whitespace",
			goModContent: "module example\n\ngo   1.22.5  \n",
			wantVersion:  "1.22.5",
			wantErr:      false,
		},
		{
			name:         "go directive first",
			goModContent: "go 1.21\nmodule example\n",
			wantVersion:  "1.21",
			wantErr:      false,
		},
		{
			name:         "missing go directive",
			goModContent: "module example\n\nrequire foo v1.0.0\n",
			wantVersion:  "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(tt.goModContent), 0o644); err != nil {
				t.Fatalf("writing go.mod: %v", err)
			}

			got, err := extractGoVersionFromDir(tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractGoVersionFromDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantVersion {
				t.Errorf("extractGoVersionFromDir() = %q, want %q", got, tt.wantVersion)
			}
		})
	}
}
