package shim

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fredrikaverpil/pocket"
	"github.com/goyek/goyek/v3"
)

func TestCalculatePocketDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		context string
		want    string
	}{
		{
			name:    "root context",
			context: ".",
			want:    ".pocket",
		},
		{
			name:    "single depth",
			context: "tests",
			want:    "../.pocket",
		},
		{
			name:    "two levels deep with forward slashes",
			context: "tests/integration",
			want:    "../../.pocket",
		},
		{
			name:    "three levels deep",
			context: "a/b/c",
			want:    "../../../.pocket",
		},
		{
			name:    "single directory with different name",
			context: "submodule",
			want:    "../.pocket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Use forward slash separator (Posix).
			got := calculatePocketDir(tt.context, "/")
			if got != tt.want {
				t.Errorf("calculatePocketDir(%q, \"/\") = %q, want %q", tt.context, got, tt.want)
			}
		})
	}
}

func TestCalculatePocketDir_PathSeparators(t *testing.T) {
	t.Parallel()

	// Test that output uses the correct path separator based on the parameter.
	tests := []struct {
		name    string
		context string
		pathSep string
		want    string
	}{
		{
			name:    "forward slash separator",
			context: "tests",
			pathSep: "/",
			want:    "../.pocket",
		},
		{
			name:    "backslash separator",
			context: "tests",
			pathSep: "\\",
			want:    "..\\.pocket",
		},
		{
			name:    "forward slash separator deep",
			context: "a/b/c",
			pathSep: "/",
			want:    "../../../.pocket",
		},
		{
			name:    "backslash separator deep",
			context: "a/b/c",
			pathSep: "\\",
			want:    "..\\..\\..\\.pocket",
		},
		{
			name:    "mixed input slashes with forward output",
			context: "a/b\\c",
			pathSep: "/",
			want:    "../../../.pocket",
		},
		{
			name:    "mixed input slashes with backslash output",
			context: "a/b\\c",
			pathSep: "\\",
			want:    "..\\..\\..\\.pocket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := calculatePocketDir(tt.context, tt.pathSep)
			if got != tt.want {
				t.Errorf("calculatePocketDir(%q, %q) = %q, want %q", tt.context, tt.pathSep, got, tt.want)
			}
		})
	}
}

func TestGenerateWithRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         pocket.Config
		wantShims      []string          // Relative paths to expected shim files.
		wantContexts   map[string]string // Map of shim path to expected POK_CONTEXT value.
		wantPocketDirs map[string]string // Map of shim path to expected POK_DIR value.
	}{
		{
			name: "root only config",
			config: pocket.Config{
				Go: &pocket.GoConfig{
					Modules: map[string]pocket.GoModuleOptions{
						".": {},
					},
				},
			},
			wantShims: []string{"pok"},
			wantContexts: map[string]string{
				"pok": ".",
			},
			wantPocketDirs: map[string]string{
				"pok": ".pocket",
			},
		},
		{
			name: "root and submodule",
			config: pocket.Config{
				Go: &pocket.GoConfig{
					Modules: map[string]pocket.GoModuleOptions{
						".":     {},
						"tests": {SkipLint: true},
					},
				},
			},
			wantShims: []string{
				"pok",
				filepath.Join("tests", "pok"),
			},
			wantContexts: map[string]string{
				"pok":                         ".",
				filepath.Join("tests", "pok"): "tests",
			},
			wantPocketDirs: map[string]string{
				"pok":                         ".pocket",
				filepath.Join("tests", "pok"): "../.pocket", // Always forward slashes for bash.
			},
		},
		{
			name: "multiple language configs",
			config: pocket.Config{
				Go: &pocket.GoConfig{
					Modules: map[string]pocket.GoModuleOptions{
						".": {},
					},
				},
				Lua: &pocket.LuaConfig{
					Modules: map[string]pocket.LuaModuleOptions{
						"scripts": {},
					},
				},
			},
			wantShims: []string{
				"pok",
				filepath.Join("scripts", "pok"),
			},
			wantContexts: map[string]string{
				"pok":                           ".",
				filepath.Join("scripts", "pok"): "scripts",
			},
			wantPocketDirs: map[string]string{
				"pok":                           ".pocket",
				filepath.Join("scripts", "pok"): "../.pocket", // Always forward slashes for bash.
			},
		},
		{
			name: "custom shim name",
			config: pocket.Config{
				Shim: &pocket.ShimConfig{
					Name:  "build",
					Posix: true,
				},
				Go: &pocket.GoConfig{
					Modules: map[string]pocket.GoModuleOptions{
						".": {},
					},
				},
			},
			wantShims: []string{"build"},
			wantContexts: map[string]string{
				"build": ".",
			},
			wantPocketDirs: map[string]string{
				"build": ".pocket",
			},
		},
		{
			name: "deeply nested context",
			config: pocket.Config{
				Go: &pocket.GoConfig{
					Modules: map[string]pocket.GoModuleOptions{
						".":     {},
						"a/b/c": {},
					},
				},
			},
			wantShims: []string{
				"pok",
				filepath.Join("a", "b", "c", "pok"),
			},
			wantContexts: map[string]string{
				"pok":                               ".",
				filepath.Join("a", "b", "c", "pok"): "a/b/c",
			},
			wantPocketDirs: map[string]string{
				"pok":                               ".pocket",
				filepath.Join("a", "b", "c", "pok"): "../../../.pocket", // Always forward slashes for bash.
			},
		},
		{
			name: "custom tasks context",
			config: pocket.Config{
				Custom: map[string][]goyek.Task{
					".":      nil, // Root custom tasks.
					"deploy": nil, // Custom tasks in deploy folder.
				},
			},
			wantShims: []string{
				"pok",
				filepath.Join("deploy", "pok"),
			},
			wantContexts: map[string]string{
				"pok":                          ".",
				filepath.Join("deploy", "pok"): "deploy",
			},
			wantPocketDirs: map[string]string{
				"pok":                          ".pocket",
				filepath.Join("deploy", "pok"): "../.pocket", // Always forward slashes for bash.
			},
		},
		{
			name: "windows shim enabled",
			config: pocket.Config{
				Shim: &pocket.ShimConfig{
					Posix:   true,
					Windows: true,
				},
				Go: &pocket.GoConfig{
					Modules: map[string]pocket.GoModuleOptions{
						".": {},
					},
				},
			},
			wantShims: []string{"pok", "pok.cmd"},
			wantContexts: map[string]string{
				"pok":     ".",
				"pok.cmd": ".",
			},
			wantPocketDirs: map[string]string{
				"pok":     ".pocket",
				"pok.cmd": ".pocket",
			},
		},
		{
			name: "powershell shim enabled",
			config: pocket.Config{
				Shim: &pocket.ShimConfig{
					Posix:      true,
					PowerShell: true,
				},
				Go: &pocket.GoConfig{
					Modules: map[string]pocket.GoModuleOptions{
						".": {},
					},
				},
			},
			wantShims: []string{"pok", "pok.ps1"},
			wantContexts: map[string]string{
				"pok":     ".",
				"pok.ps1": ".",
			},
			wantPocketDirs: map[string]string{
				"pok":     ".pocket",
				"pok.ps1": ".pocket",
			},
		},
		{
			name: "all shim types enabled",
			config: pocket.Config{
				Shim: &pocket.ShimConfig{
					Posix:      true,
					Windows:    true,
					PowerShell: true,
				},
				Go: &pocket.GoConfig{
					Modules: map[string]pocket.GoModuleOptions{
						".": {},
					},
				},
			},
			wantShims: []string{"pok", "pok.cmd", "pok.ps1"},
			wantContexts: map[string]string{
				"pok":     ".",
				"pok.cmd": ".",
				"pok.ps1": ".",
			},
			wantPocketDirs: map[string]string{
				"pok":     ".pocket",
				"pok.cmd": ".pocket",
				"pok.ps1": ".pocket",
			},
		},
		{
			name: "windows only - no posix",
			config: pocket.Config{
				Shim: &pocket.ShimConfig{
					Windows: true,
				},
				Go: &pocket.GoConfig{
					Modules: map[string]pocket.GoModuleOptions{
						".": {},
					},
				},
			},
			wantShims: []string{"pok.cmd"},
			wantContexts: map[string]string{
				"pok.cmd": ".",
			},
			wantPocketDirs: map[string]string{
				"pok.cmd": ".pocket",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a temporary directory for testing.
			tmpDir := t.TempDir()

			// Create the .pocket directory with a minimal go.mod.
			pocketDir := filepath.Join(tmpDir, ".pocket")
			if err := os.MkdirAll(pocketDir, 0o755); err != nil {
				t.Fatalf("creating .pocket directory: %v", err)
			}

			goMod := "module pocket\n\ngo 1.25.5\n"
			if err := os.WriteFile(filepath.Join(pocketDir, "go.mod"), []byte(goMod), 0o644); err != nil {
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

				// Verify executable permissions (skip on Windows as it doesn't have Unix permissions).
				if runtime.GOOS != "windows" && info.Mode().Perm()&0o100 == 0 {
					t.Errorf("shim %q is not executable", shimPath)
				}

				// Read and verify content.
				content, err := os.ReadFile(fullPath)
				if err != nil {
					t.Errorf("reading shim %q: %v", shimPath, err)
					continue
				}

				contentStr := string(content)

				// Determine the shim type based on extension.
				isBash := !strings.HasSuffix(shimPath, ".cmd") && !strings.HasSuffix(shimPath, ".ps1")
				isCmd := strings.HasSuffix(shimPath, ".cmd")
				isPs1 := strings.HasSuffix(shimPath, ".ps1")

				// Verify POK_CONTEXT.
				if wantContext, ok := tt.wantContexts[shimPath]; ok {
					var found bool
					switch {
					case isBash:
						found = strings.Contains(contentStr, `POK_CONTEXT="`+wantContext+`"`)
					case isCmd:
						found = strings.Contains(contentStr, `set "POK_CONTEXT=`+wantContext+`"`)
					case isPs1:
						found = strings.Contains(contentStr, `$PocketContext = "`+wantContext+`"`)
					}
					if !found {
						t.Errorf("shim %q: expected POK_CONTEXT=%q not found in content", shimPath, wantContext)
					}
				}

				// Verify POK_DIR.
				if wantPocketDir, ok := tt.wantPocketDirs[shimPath]; ok {
					var found bool
					// Windows shims use backslashes in paths.
					windowsPocketDir := strings.ReplaceAll(wantPocketDir, "/", "\\")
					switch {
					case isBash:
						found = strings.Contains(contentStr, `POK_DIR="`+wantPocketDir+`"`)
					case isCmd:
						found = strings.Contains(contentStr, `set "POK_DIR=`+windowsPocketDir+`"`)
					case isPs1:
						found = strings.Contains(contentStr, `$PocketDir = "`+windowsPocketDir+`"`)
					}
					if !found {
						t.Errorf("shim %q: expected POK_DIR=%q not found in content", shimPath, wantPocketDir)
					}
				}

				// Verify Go version (only for bash and powershell which include it).
				if isBash && !strings.Contains(contentStr, `GO_VERSION="1.25.5"`) {
					t.Errorf("shim %q: expected GO_VERSION=1.25.5 not found", shimPath)
				}
				if isPs1 && !strings.Contains(contentStr, `$GoVersion = "1.25.5"`) {
					t.Errorf("shim %q: expected GoVersion=1.25.5 not found", shimPath)
				}
			}
		})
	}
}

func TestGenerateWithRoot_MissingGoMod(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .pocket directory without go.mod.
	pocketDir := filepath.Join(tmpDir, ".pocket")
	if err := os.MkdirAll(pocketDir, 0o755); err != nil {
		t.Fatalf("creating .pocket directory: %v", err)
	}

	config := pocket.Config{
		Go: &pocket.GoConfig{
			Modules: map[string]pocket.GoModuleOptions{
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

	// Create .pocket directory with go.mod that has no go directive.
	pocketDir := filepath.Join(tmpDir, ".pocket")
	if err := os.MkdirAll(pocketDir, 0o755); err != nil {
		t.Fatalf("creating .pocket directory: %v", err)
	}

	goMod := "module pocket\n\nrequire example.com/foo v1.0.0\n"
	if err := os.WriteFile(filepath.Join(pocketDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	config := pocket.Config{
		Go: &pocket.GoConfig{
			Modules: map[string]pocket.GoModuleOptions{
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
			goModContent: "module example\n\ngo 1.25.5\n",
			wantVersion:  "1.25.5",
			wantErr:      false,
		},
		{
			name:         "go directive with extra whitespace",
			goModContent: "module example\n\ngo   1.25.5  \n",
			wantVersion:  "1.25.5",
			wantErr:      false,
		},
		{
			name:         "go directive first",
			goModContent: "go 1.25.5\nmodule example\n",
			wantVersion:  "1.25.5",
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
